// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cardano

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	defaultPageCount = 100
	maxWorkers       = 6
	httpTimeout      = 30 * time.Second
)

type BFClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type AssetTx struct {
	TxHash string `json:"tx_hash"`
	Block  string `json:"block"`
}

type TxUtxos struct {
	Hash    string        `json:"hash"`
	Inputs  []TxUtxoEntry `json:"inputs"`
	Outputs []TxUtxoEntry `json:"outputs"`
}

type TxUtxoEntry struct {
	Address     string       `json:"address"`
	Amount      []AmountItem `json:"amount"`
	OutputIndex *int         `json:"output_index,omitempty"`
}

type AmountItem struct {
	Unit     string `json:"unit"`
	Quantity string `json:"quantity"`
}

type Match struct {
	TxHash      string
	BlockHash   string
	OutputIndex *int
	FromAddrs   []string
	ToAddress   string
	Quantity    string
	Unit        string
}

func NewClient(apiKey, endpoint string) *BFClient {
	return &BFClient{
		baseURL: endpoint,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: httpTimeout,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}
}

func (b *BFClient) doRequest(ctx context.Context, method, path string, params url.Values) (*http.Response, error) {
	rel := path
	if params != nil {
		rel = rel + "?" + params.Encode()
	}
	u := b.baseURL + rel
	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("project_id", b.apiKey)
	req.Header.Set("Accept", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var bodyRepr struct {
			Error string `json:"error"`
			Msg   string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&bodyRepr)
		resp.Body.Close()
		return nil, fmt.Errorf("blockfrost error: status=%d url=%s err=%v", resp.StatusCode, u, bodyRepr)
	}
	return resp, nil
}

func (b *BFClient) ListAssetTxs(ctx context.Context, asset string) (<-chan AssetTx, <-chan error) {
	out := make(chan AssetTx)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errCh)
		page := 1
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}
			params := url.Values{}
			params.Set("page", strconv.Itoa(page))
			params.Set("count", strconv.Itoa(defaultPageCount))
			path := fmt.Sprintf("/assets/%s/transactions", url.PathEscape(asset))

			resp, err := b.doRequest(ctx, http.MethodGet, path, params)
			if err != nil {
				errCh <- fmt.Errorf("assets transactions request failed: %w", err)
				return
			}

			body, err := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			if err != nil {
				errCh <- fmt.Errorf("failed to read body: %w", err)
				return
			}

			var items []AssetTx
			err = json.Unmarshal(body, &items)
			if err != nil {
				errCh <- fmt.Errorf("decode asset txs page %d failed: %w", page, err)
				return
			}

			if len(items) == 0 {
				return
			}

			for _, it := range items {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case out <- it:
				}
			}

			if len(items) < defaultPageCount {
				return
			}
			page++
			time.Sleep(120 * time.Millisecond)
		}
	}()
	return out, errCh
}

func (b *BFClient) GetTxUtxos(ctx context.Context, txHash string) (*TxUtxos, error) {
	path := fmt.Sprintf("/txs/%s/utxos", url.PathEscape(txHash))
	resp, err := b.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result TxUtxos
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (b *BFClient) FindTxsToAddressForAsset(ctx context.Context, asset, destAddress string) ([]Match, error) {
	assetTxsCh, errCh := b.ListAssetTxs(ctx, asset)
	type job struct{ assetTx AssetTx }
	jobs := make(chan job)
	results := make(chan Match)
	var wg sync.WaitGroup

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := range jobs {
				utxos, err := b.GetTxUtxos(ctx, j.assetTx.TxHash)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					fmt.Fprintf(os.Stderr, "warning: worker %d: failed to fetch utxos for tx %s: %v\n", workerID, j.assetTx.TxHash, err)
					time.Sleep(300 * time.Millisecond)
					continue
				}

				// sender(inputs)
				fromSet := make(map[string]struct{})
				for _, inp := range utxos.Inputs {
					fromSet[inp.Address] = struct{}{}
				}
				fromAddrs := make([]string, 0, len(fromSet))
				for a := range fromSet {
					fromAddrs = append(fromAddrs, a)
				}

				// outputs of destination
				for _, out := range utxos.Outputs {
					if out.Address != destAddress {
						continue
					}
					for _, amt := range out.Amount {
						if amt.Unit == asset {
							results <- Match{
								TxHash:      j.assetTx.TxHash,
								BlockHash:   j.assetTx.Block,
								OutputIndex: out.OutputIndex,
								FromAddrs:   fromAddrs,
								ToAddress:   out.Address,
								Quantity:    amt.Quantity,
								Unit:        amt.Unit,
							}
						}
					}
				}
				time.Sleep(80 * time.Millisecond)
			}
		}(i)
	}

	var collectorErr error
	doneCollect := make(chan struct{})
	go func() {
		defer close(doneCollect)
		defer close(jobs)
		for tx := range assetTxsCh {
			select {
			case <-ctx.Done():
				collectorErr = ctx.Err()
				return
			default:
			}
			jobs <- job{assetTx: tx}
		}
	}()

	go func() {
		<-doneCollect
		wg.Wait()
		close(results)
	}()

	var matches []Match
collectLoop:
	for {
		select {
		case e := <-errCh:
			if e != nil {
				return nil, e
			}
		case m, ok := <-results:
			if !ok {
				break collectLoop
			}
			matches = append(matches, m)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if collectorErr != nil {
		return nil, collectorErr
	}
	return matches, nil
}
