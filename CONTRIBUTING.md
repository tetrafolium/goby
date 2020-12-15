First of all, thank you for trying to contribute goby, any contribution will be appreciated 😁

The following is the guideline (not rules) for contributing goby, I suggest you read them all before you start your contribution.

If you are very interested in `Goby` or planning contribute `Goby` frequently, please contact me directly.

## What to contribute

- Any issues you see, and if you think the ticket is confusing, please open an issue or ask me on [slack](https://goby-slack-invite.herokuapp.com/).
- Any grammar error in readme, wiki, and code comments...etc.
- Any issues listed in goby's [Go Report Card](https://goreportcard.com/report/github.com/goby-lang/goby).
- Play around goby and report any bug you find.
- Write benchmarks for goby (we really need this and really haven't had time to do it yet 😢)
- Help us document built in class and libraries' api, see the [guideline](https://github.com/tetrafolium/goby/wiki/Documenting-Goby-Code)


#### If you're interested in lexeing/parsing, please check `token`, `lexer`, `ast` and `parser` packages

#### If you're interested in compiler, check [bytecode specifications](https://github.com/tetrafolium/goby/wiki/Bytecode-Instruction-specs) and bytecode package's [tests](https://github.com/tetrafolium/goby/blob/master/bytecode/generator_test.go) for some compiled examples.

#### If you're interested in VM's structure, please contact me directly since a lot of things haven't been documented yet.

#### If you're a Ruby developer:
  - you can start with adding methods to built in classes like [`Array`](https://github.com/tetrafolium/goby/blob/master/vm/array.go) or [`Hash`](https://github.com/tetrafolium/goby/blob/master/vm/hash.go) using `Golang`. And here's a [guideline](https://github.com/tetrafolium/goby/wiki/Contibuting-a-Method) for contributing built in methods.
  - you can also porting Ruby's standard lib using `Goby` (not Go), see [lib directory](https://github.com/tetrafolium/goby/tree/master/lib/net). You'll feel like you're just writing plain Ruby 😄

#### If you want to propose a feature, just open an issue with `[feature request]` prefix on title.

**Note**:
  - Before sending PR, you should perform `make test` on the root directory of the project to perform all tests (`go test` works only against goby.go file and will be incomplete for the test).
  - DB library tests requires Postgresql to be opened and export port `5432`


## Setup Environment


### `$GOBY_ROOT`

By default Goby finds standard libs in `/usr/local/goby` when you install it via homebrew.
But if you want to develop Goby or you installed Goby from source, you might want to set `$GOBY_ROOT` to Goby's project root so you can use latest libs.
Add the following line to your shell config file, either `~/.bashrc`, `~/.bash_profile`, or `~/.zshrc` if you're using zsh.

```
export GOBY_ROOT=$GOPATH/src/github.com/tetrafolium/goby
```

The most common messages you'll see if you do not set `$GOBY_ROOT` right are 'library not found'. For example:

```ruby
require 'net/http'
# => Internal Error: open lib/net/http/response.gb: no such file or directory
```



## To Run Tests

If you want to run Go tests, you can run:

```
$ go test PKG_NAME -run TestName
```

For example, this will run any tests in the `vm` package that matches `TestIncludeFail` with their names:

```
$ go test ./vm -run TestIncludeFail
```

You can run `go help test` to see more options.

And if you want to run Goby tests, you can use:

```
$ goby test specs
```

or if you want to test it against your latest changes in Goby source code, you can use

```
$ go run goby.go test specs
```

But we haven't support running single Goby test at the moment (contribution is welcomed 😉).



