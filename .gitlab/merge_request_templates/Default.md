## Merge Request

### Description

<!-- Provide a brief summary of the changes introduced by this merge request also linking the related GitLab issue. Explain wether the changes are considered a breaking change, a new feature, or a bug fix. -->


### Type of Change

<!-- Choose the appropriate type for the changes -->

- [ ] feat: A new feature for the user or a significant enhancement
- [ ] fix: A bug fix
- [ ] refactor: Code changes that neither fix a bug nor add a feature
- [ ] perf: Performance improvements
- [ ] test: Adding or modifying tests
- [ ] style: Code style changes (e.g., formatting)
- [ ] docs: Documentation changes
- [ ] revert: Reverting a previous commit
- [ ] build: Changes that affect the build system or external dependencies (e.g. npm, docker, nexus)
- [ ] chore: Routine tasks, maintenance, or general refactoring
- [ ] ci: Changes to the project's Continuous Integration (CI) configuration
- [ ] release: Version number changes, typically associated with creating a new release
- [ ] other (please specify):


### Semantic Versioning

<!-- Indicate the impact of the changes on the version number -->
<!-- Choose ONLY ONE box and do not change the description -->

- [ ] This MR introduces a **breaking change**. (If yes, use the prefix `BREAKING CHANGE: <merge request title>` and ensure version bumps to `major`.)
- [ ] This MR adds a new **feature** in a backwards-compatible manner. (If yes, use the prefix `feat: <merge request title>` and ensure version bumps to `minor`.)
- [ ] This MR only includes **bug fixes or patches** in a backwards-compatible manner. (If yes, use the prefix - `fix: <merge request title>` and ensure version bumps to `patch`.)
- [ ] This MR does not necessitate a change in the version, since it contains routine maintenance, housekeeping, or improvements that don't affect the external API or user experience. (If yes, use the prefix that fits better e.g., `build/chore/ci` - `chore: <merge request title>` and ensure version doesn't bump.)


### Specification and Related Issues
<!-- Link to the specification and any related issues, documentation, MRs (e.g., Issue #123, Fix #456) -->


### Checklist

<!-- Check the tasks that are done. Ideally, everything should be marked. -->

Testing Aspects:
- [ ] The changes have been thoroughly tested.
- [ ] Unit tests were added or modified to cover the changes.
- [ ] Gherkin features are written (if applicable) according to these [instructions](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/specification_and_documentation/README.md#gherkin-feature-file)
- [ ] All tests pass successfully.

Security Aspects:
- [ ] This MR doesn’t include any hardcoded secrets/passwords and doesn't store any passwords in the repository. (Use Vault and ask for support from the Security Team if needed.)
- [ ] The .gitignore file excludes files as .env files, or any other sensitive files/directories. (Examples for .gitignore files [here](https://github.com/github/gitignore).)
- [ ] Consider potential security abuses of the feature and test accordingly. If you can't fix the security problem, create a new issue to address this problem and link it in this MR.

Documentation Aspects:
- [ ] This MR is linked to any relevant GitLab issues.
- [ ] This MR is linked to the specification and complies with it.
- [ ] Documentation has been updated or added according to the [documentation procedure](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/specification_and_documentation/README.md).
- [ ] Swagger API documentation is updated (if applicable).

Other Aspects:
- [ ] The code follows the project's coding standards and has been checked by the code_quality stage in the pipeline.
- [ ] The source branch is up-to-date with the target branch.
- [ ] This MR is assigned to the appropriate reviewer(s).


### Additional Information

<!-- Provide any additional information or context that might be helpful for reviewers -->

