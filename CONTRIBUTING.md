# Contributing to HTTP Broadcast

:+1::tada: First off, thanks for taking the time to contribute! :tada::+1:

The following is a set of guidelines for contributing to HTTP Broadcast project.
These are mostly guidelines, not rules. Use your best judgment, and feel free to
propose changes to this document in a pull request.

## Install and run the project locally

Clone the project:

    $ git clone https://github.com/jderusse/http-broadcast
    $ cd http-broadcast

Run the application:

    $ go run .

To run the test suite:

    $ go test ./... -race

To lint your code:

    $ # go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
    $ golangci-lint run --fix  ./...

## Pull Request Process

Before submitting a Pull Request, make sure that:

* Tests are green - including lint.
* You add valid test cases.
* You document the new behaviors
