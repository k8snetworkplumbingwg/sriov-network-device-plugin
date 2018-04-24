# SRIOV Network Device plugin

* [How to Contribute](#contributing-code)
* [Coding Style](#contributing-code)
* [Contributing Code](#contributing-code)
* [Tools](#Tools)

## How to Contribute

SRIOV Network Device plugin is [Apache 2.0 licensed](LICENSE) and accepts contributions via GitHubpull requests. This document outlines some of the conventions on development workflow, commit message formatting, contact points and other resources to make it easier to get your contribution accepted.

## Coding Style

Please follows the standard formatting recommendations and language idioms set out in [Effective Go](https://golang.org/doc/effective_go.html) and in the [Go Code Review Comments wiki](https://github.com/golang/go/wiki/CodeReviewComments).

## Contributing Code

We always encourage the contribution for the community project. We like to collaborate with various stake holder on this project. We ask developer to keep following guidelines in mind before the contribution.

* Make sure to create an [Issue](https://github.com/intel/sriov-network-device-plugin/issues) for bug fix or the feature request.  
* **For bugs**: For the bug fixes, please follow the issue template format while creating a issue.  If you have already found a fix, feel free to submit a Pull Request referencing the Issue you created. Include the `Fixes #` syntax to link it to the issue you're addressing.
* **For feature requests**, For the feature requests, please follow the issue template format while creating a feature requests. We want to improve upon SRIOV Network device plugin incrementally which means small changes or features at a time.
  * Please make sure each PR are compiling or passed by Travis.
  * In order to ensure your PR can be reviewed in a timely manner, please keep PRs small 
 
Once you're ready to contribute code back to this repo, start with these steps:
* Fork the appropriate sub-projects that are affected by your change.
* Clone the fork to your machine:

```
$ git clone https://github.com/intel/sriov-network-device-plugin.git
```

* Create a topic branch with prefix `dev/` for your change and checkout that branch:

```
$ git checkout -b dev/some-topic-branch
```
* Make your changes to the code and add tests to cover contributed code.
* Run `./build.sh` to validate it builds and will not break current functionality.
* Commit your changes and push them to your fork.
* Open a pull request for the appropriate project.
* Contributors will review your pull request, suggest changes, run tests and eventually merge or close the request.

> We encourage contributor to test SRIOV Network Device plugin with various NICs to check the compatibility.
> 
## Tools
The project uses the Slack chat:
- Slack: #[Intel-SDSG-slack](https://intel-corp.herokuapp.com/) channel on slack
- Please contact contributors for issues and PR reviews.
- Can't comment much how Slack communication improve the development workflow and ability to maintain a good system of communication.