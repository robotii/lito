# Lito Programming Language

**Lito** is an scripting language heavily influenced by **Ruby** and **Smalltalk**, as well as the implementation language, **Go**.


## Installation

If you'd like to contribute, or are just interested in seeing the source, simply clone this repository.

To create the lito executable, run the following command in the project root.

```
go build ./cmd/lito
```

This will download the dependencies and compile Lito. Currently only github.com/chzyer/readline is a required dependency.

This should produce the `lito` executable in the root of the project

To start Lito interactively, simply run the following command:

```
./lito
```

To try out the examples add the path to the file on the command line.

```
./lito examples/error.lito
```
