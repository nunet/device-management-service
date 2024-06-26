# Introduction

Jobs package will manage local jobs and their allocation, including relation to execution environments, etc. It will manage jobs running with whatever executor (container, VM, Direct_exe, Java etc). It's responsible for parsing job specs, comparison and overall local job management.

## Types and data models

### Allocation

_proposed by @kabir.kbr; date: 2024-04-22_

As per initial [specification of NuNet ontology / nomenclature](https://nunet.gitlab.io/research/blog/posts/ontology-and-nomenclature/#actors), `Allocation` extend the `models.Actor` interface and by that makes the running job a first class citizen of NuNet's Actor model, so being able to send and receive messages and maintain state.

* type definition: [nunet/open-api/platform-data-model/device-management-service/jobs/allocation.go]((https://gitlab.com/nunet/open-api/platform-data-model/device-management-service/jobs/allocation.go);
