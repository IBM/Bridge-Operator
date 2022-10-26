# Contributing

## Contribution Terms and License

The documentation and benchmarking of The Bridge Operator is contained in this repository. To contribute
to this project or any of the elements of The Bridge Operator we recommend you start by reading this
contributing guide.

## Contributor License Agreement

Before you can submit any code we need all contributors to sign a
contributor license agreement. By signing a contributor license
agreement (CLA) you're basically just attesting to the fact
that you are the author of the contribution and that you're freely
contributing it under the terms of the MIT License.

When you contribute to the Bridge Operator project with a new pull request,
a bot will evaluate whether you have signed the CLA. If required, the
bot will comment on the pull request, including a link to accept the
agreement. The [individual CLA](./iCLA.md) document is available for review in this repo.

## Contributing to Bridge Operator codebase

If you would like to contribute to the package, we recommend the following development setup.

1. Create a copy of the [repository](https://github.com/Bridge-Operator) via the "_Fork_" button.

2. Clone the Bridge Operator repository:

    ```sh
    git clone git@github.com:${GH_ACCOUNT_OR_ORG}/Bridge-Operator.git
    ```

3. Add remote Bridge-Operator repo as an "upstream" in your local repo, so you can check/update remote changes.

   ```sh
   git remote add upstream git@github.com:Bridge-Operator/Bridge-Operator.git
   ```

4. Create a dedicated branch:

    ```sh
    cd Bridge-Operator
    git checkout -b a-super-nice-feature-we-all-need
    ```

5. Create and activate a dedicated conda environment:

    ```sh
    conda env create -f conda.yml
    conda activate Bridge-Operator
    ```



9. From your fork, open a pull request via the "_Contribute_" button, the maintainers will be happy to review it.

