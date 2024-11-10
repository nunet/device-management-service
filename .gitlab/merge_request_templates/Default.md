## Merge Request

### Description

<!-- Provide a brief summary of the changes introduced by this merge request also linking the related GitLab issue. Explain wether the changes are considered a breaking change, a new feature, or a bug fix. -->

### Type of Change

<!-- Choose ONE box to indicate the impact of the changes -->

- [ ] This MR introduces a **breaking change**.
- [ ] This MR adds a new **feature** in a backwards-compatible manner.
- [ ] This MR only includes **bug fixes or patches** in a backwards-compatible manner.
- [ ] This MR contains routine maintenance, housekeeping, or improvements that don't affect the external API or user experience.

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

Other Aspects:
- [ ] The code follows the project's coding standards and has been checked by the code_quality stage in the pipeline.
- [ ] The source branch is up-to-date with the target branch.
- [ ] This MR is assigned to the appropriate reviewer(s).


### Additional Information

<!-- Provide any additional information or context that might be helpful for reviewers -->

