# go-re2

go-re2 is a drop-in replacement for the standard library [regexp][1] package which uses the C++
[re2][2] library for improved performance with large inputs or complex expressions. By default,
re2 is packaged as a WebAssembly module and accessed with the pure Go runtime, [wazero][3].
This means that it is compatible with any Go application, regardless of availability of cgo.

Note that if your regular expressions or input are small, this library is slower than the
standard library. You will generally "know" if your application requires high performance for
complex regular expressions, for example in security filtering software. If you do not know
your app has such needs, you should turn away now.

## Behavior differences

The library is almost fully compatible with the standard library regexp package, with just a few
behavior differences. These are likely corner cases that don't affect typical applications. It is
best to confirm them before proceeding.

- Invalid utf-8 strings are treated differently. The standard library silently replaces invalid utf-8
  with the unicode replacement character. This library will stop consuming strings when encountering
  invalid utf-8.

  - `experimental.CompileLatin1` can be used to match against non-utf8 strings

- `reflect.DeepEqual` cannot compare `Regexp` objects.

Continue to use the standard library if your usage would match any of these.

Searching this codebase for `// GAP` will allow finding tests that have been tweaked to demonstrate
behavior differences.

## API differences

All APIs found in `regexp` are available except

- `*Reader`: re2 does not support streaming input

Note that unlike many packages that wrap C++ libraries, there is no added `Close` type of method.
See the [rationale](./RATIONALE.md) for more details.

### Experimental APIs

The [experimental](./experimental) package contains APIs not part of standard `regexp` that are
incubating. They may in the future be moved to stable packages. The experimental package does not
provide any guarantee of API stability even across minor version updates.

## Usage

go-re2 is a standard Go library package and can be added to a go.mod file. It will work fine in
any Go project. See [below](#tinygo) for notes about TinyGo support.

```
go get github.com/wasilibs/go-re2
```

Because the library is a drop-in replacement for the standard library, an import alias can make
migrating code to use it simple.

```go
import "regexp"
```

can be changed to

```go
import regexp "github.com/wasilibs/go-re2"
```

### cgo

This library also supports opting into using cgo to wrap re2 instead of using WebAssembly. This
requires having re2 installed and available via `pkg-config` on the system. The build tag `re2_cgo`
can be used to enable cgo support.

#### Ubuntu

On Ubuntu install the gcc tool chain and the re2 library as follows:

```bash
sudo apt install build-essential
sudo apt-get install -y libre2-dev
```

#### Alpine

On Alpine install the gcc tool chain and the re2 library as follows:

```bash
apk add build-base pkgconfig re2-dev
```

#### Windows

On Windows start by installing [MSYS2][8]. Then open the MINGW64 terminal and install the gcc toolchain and re2 via pacman:

```bash
pacman -S mingw-w64-x86_64-gcc
pacman -S mingw-w64-x86_64-re2
pacman -S mingw-w64-x86_64-pkg-config
```

If you want to run the resulting exe program outside the MINGW64 terminal you need to add a path to the MinGW-w64 libraries to the PATH environmental variable (adjust as needed for your system):

```cmd
SET PATH=C:\msys64\mingw64\bin;%PATH%
```

#### MacOS

On Mac start by installing [homebrew][9] including installation of the command line tools. Then install re2 via brew:

```bash
brew install re2
```

### TinyGo

This project began as a way to use re2 with TinyGo WASI projects. However, recent versions of re2 have reworked
their build, notably depending on absl which requires threads support and breaks compatibility with TinyGo programs.
To stay up-to-date with re2, after much time this project has removed building of TinyGo-specific Wasm artifacts.
Build tags to enable cgo codepaths for TinyGo are kept to provide best-effort support with projects bringing their
own Wasm, with the only known usage currently in [coraza-wasilibs][10].

## Performance

Benchmarks are run against every commit in the [bench][4] workflow. GitHub action runners are highly
virtualized and do not have stable performance across runs, but the relative numbers within a run
should still be somewhat, though not precisely, informative.

### wafbench

wafbench tests the performance of replacing the regex operator of the OWASP [CoreRuleSet][5] and
[Coraza][6] implementation with this library. This benchmark is considered a real world performance
test, with the regular expressions being real ones used in production. Security filtering rules
often have highly complex expressions.

One run looks like this

```
name \ time/op     build/wafbench_stdlib.txt  build/wafbench.txt  build/wafbench_cgo.txt
WAF/FTW-4                         21.72 ± ∞ ¹    21.17 ± ∞ ¹   -2.55% (p=0.008 n=5)    19.51 ± ∞ ¹  -10.19% (p=0.008 n=5)
WAF/POST/1-4                     2.308m ± ∞ ¹   2.727m ± ∞ ¹  +18.18% (p=0.008 n=5)   2.479m ± ∞ ¹   +7.44% (p=0.008 n=5)
WAF/POST/1000-4                 15.399m ± ∞ ¹   4.734m ± ∞ ¹  -69.26% (p=0.008 n=5)   3.742m ± ∞ ¹  -75.70% (p=0.008 n=5)
WAF/POST/10000-4                150.65m ± ∞ ¹   19.90m ± ∞ ¹  -86.79% (p=0.008 n=5)   13.47m ± ∞ ¹  -91.06% (p=0.008 n=5)
WAF/POST/100000-4               1511.3m ± ∞ ¹   171.7m ± ∞ ¹  -88.64% (p=0.008 n=5)   110.4m ± ∞ ¹  -92.69% (p=0.008 n=5)
```

`FTW` is the time to run the standard CoreRuleSet test framework. The performance of this
library with WebAssembly, wafbench.txt, shows a slight improvement over the standard library
in this baseline test case.

The FTW test suite will issue many requests with various payloads, generally somewhat small.
The `POST` tests show the same ruleset applied to requests with payload sizes as shown, in bytes.
We see that only with the absolute smallest payload of 1 byte does the standard library perform
a bit better than this library. For any larger size, even a fairly typical 1KB, go-re2
greatly outperforms.

cgo seems to offer about a 30% improvement on WebAssembly in this library. Many apps may accept
the somewhat slower performance in exchange for the build and deployment flexibility of
WebAssembly but either option will work with no changes to the codebase.

### Microbenchmarks

Microbenchmarks are the same as included in the Go standard library. Full results can be
viewed in the workflow, a sample of results for one run looks like this

```
name \ time/op                  build/bench_stdlib.txt  build/bench.txt   build/bench_cgo.txt
Find-4                                     162.3n ± ∞ ¹       1057.0n ± ∞ ¹    +551.26% (p=0.008 n=5)     397.5n ± ∞ ¹   +144.92% (p=0.008 n=5)
Compile/Onepass-4                          4.017µ ± ∞ ¹       23.585µ ± ∞ ¹    +487.13% (p=0.008 n=5)     7.803µ ± ∞ ¹    +94.25% (p=0.008 n=5)
Compile/Medium-4                           9.042µ ± ∞ ¹       45.105µ ± ∞ ¹    +398.84% (p=0.008 n=5)    13.346µ ± ∞ ¹    +47.60% (p=0.008 n=5)
Compile/Hard-4                             66.52µ ± ∞ ¹       272.58µ ± ∞ ¹    +309.80% (p=0.008 n=5)     94.49µ ± ∞ ¹    +42.06% (p=0.008 n=5)
Match/Easy0/16-4                           3.721n ± ∞ ¹      615.300n ± ∞ ¹  +16435.88% (p=0.008 n=5)   155.900n ± ∞ ¹  +4089.73% (p=0.008 n=5)
Match/Easy0/32-4                           40.41n ± ∞ ¹       620.10n ± ∞ ¹   +1434.52% (p=0.008 n=5)    155.00n ± ∞ ¹   +283.57% (p=0.008 n=5)
Match/Easy0/1K-4                           242.3n ± ∞ ¹        689.7n ± ∞ ¹    +184.65% (p=0.008 n=5)     154.8n ± ∞ ¹    -36.11% (p=0.008 n=5)
Match/Easy0/32K-4                         3279.0n ± ∞ ¹       1384.0n ± ∞ ¹     -57.79% (p=0.008 n=5)     154.0n ± ∞ ¹    -95.30% (p=0.008 n=5)
Match/Easy0/1M-4                        240196.0n ± ∞ ¹      42954.0n ± ∞ ¹     -82.12% (p=0.008 n=5)     155.0n ± ∞ ¹    -99.94% (p=0.008 n=5)
Match/Easy0/32M-4                      7942195.0n ± ∞ ¹    1359859.0n ± ∞ ¹     -82.88% (p=0.008 n=5)     155.4n ± ∞ ¹   -100.00% (p=0.008 n=5)
Match/Easy0i/16-4                          3.723n ± ∞ ¹      589.600n ± ∞ ¹  +15736.69% (p=0.008 n=5)   155.800n ± ∞ ¹  +4084.80% (p=0.008 n=5)
Match/Easy0i/32-4                          681.2n ± ∞ ¹        593.3n ± ∞ ¹     -12.90% (p=0.008 n=5)     154.3n ± ∞ ¹    -77.35% (p=0.008 n=5)
Match/Easy0i/1K-4                        20325.0n ± ∞ ¹        667.6n ± ∞ ¹     -96.72% (p=0.008 n=5)     155.4n ± ∞ ¹    -99.24% (p=0.008 n=5)
Match/Easy0i/32K-4                      913391.0n ± ∞ ¹       1368.0n ± ∞ ¹     -99.85% (p=0.008 n=5)     155.2n ± ∞ ¹    -99.98% (p=0.008 n=5)
Match/Easy0i/1M-4                     29087522.0n ± ∞ ¹      42975.0n ± ∞ ¹     -99.85% (p=0.008 n=5)     155.5n ± ∞ ¹   -100.00% (p=0.008 n=5)
Match/Easy0i/32M-4                   929572314.0n ± ∞ ¹    1362975.0n ± ∞ ¹     -99.85% (p=0.008 n=5)     155.7n ± ∞ ¹   -100.00% (p=0.008 n=5)
Match/Easy1/16-4                           3.721n ± ∞ ¹      591.000n ± ∞ ¹  +15782.83% (p=0.008 n=5)   156.600n ± ∞ ¹  +4108.55% (p=0.008 n=5)
Match/Easy1/32-4                           37.39n ± ∞ ¹       595.30n ± ∞ ¹   +1492.14% (p=0.008 n=5)    155.40n ± ∞ ¹   +315.62% (p=0.008 n=5)
Match/Easy1/1K-4                           517.5n ± ∞ ¹        685.1n ± ∞ ¹     +32.39% (p=0.008 n=5)     156.4n ± ∞ ¹    -69.78% (p=0.008 n=5)
Match/Easy1/32K-4                        25905.0n ± ∞ ¹       1342.0n ± ∞ ¹     -94.82% (p=0.008 n=5)     158.2n ± ∞ ¹    -99.39% (p=0.008 n=5)
Match/Easy1/1M-4                        929177.0n ± ∞ ¹      42940.0n ± ∞ ¹     -95.38% (p=0.008 n=5)     155.1n ± ∞ ¹    -99.98% (p=0.008 n=5)
Match/Easy1/32M-4                     29798707.0n ± ∞ ¹    1362212.0n ± ∞ ¹     -95.43% (p=0.008 n=5)     155.6n ± ∞ ¹   -100.00% (p=0.008 n=5)
Match/Medium/16-4                          3.723n ± ∞ ¹      590.900n ± ∞ ¹  +15771.61% (p=0.008 n=5)   155.000n ± ∞ ¹  +4063.31% (p=0.008 n=5)
Match/Medium/32-4                          566.8n ± ∞ ¹        595.2n ± ∞ ¹      +5.01% (p=0.008 n=5)     154.9n ± ∞ ¹    -72.67% (p=0.008 n=5)
Match/Medium/1K-4                        20063.0n ± ∞ ¹        669.7n ± ∞ ¹     -96.66% (p=0.008 n=5)     155.3n ± ∞ ¹    -99.23% (p=0.008 n=5)
Match/Medium/32K-4                      924929.0n ± ∞ ¹       1340.0n ± ∞ ¹     -99.86% (p=0.008 n=5)     155.8n ± ∞ ¹    -99.98% (p=0.008 n=5)
Match/Medium/1M-4                     29406989.0n ± ∞ ¹      42947.0n ± ∞ ¹     -99.85% (p=0.008 n=5)     154.6n ± ∞ ¹   -100.00% (p=0.008 n=5)
Match/Medium/32M-4                   963966642.0n ± ∞ ¹    1363441.0n ± ∞ ¹     -99.86% (p=0.008 n=5)     154.6n ± ∞ ¹   -100.00% (p=0.008 n=5)
Match/Hard/16-4                            3.744n ± ∞ ¹      596.000n ± ∞ ¹  +15818.80% (p=0.008 n=5)   155.600n ± ∞ ¹  +4055.98% (p=0.008 n=5)
Match/Hard/32-4                            997.3n ± ∞ ¹        598.1n ± ∞ ¹     -40.03% (p=0.008 n=5)     155.6n ± ∞ ¹    -84.40% (p=0.008 n=5)
Match/Hard/1K-4                          30435.0n ± ∞ ¹        686.5n ± ∞ ¹     -97.74% (p=0.008 n=5)     154.7n ± ∞ ¹    -99.49% (p=0.008 n=5)
Match/Hard/32K-4                       1348825.0n ± ∞ ¹       1342.0n ± ∞ ¹     -99.90% (p=0.008 n=5)     155.7n ± ∞ ¹    -99.99% (p=0.008 n=5)
Match/Hard/1M-4                       43023861.0n ± ∞ ¹      42891.0n ± ∞ ¹     -99.90% (p=0.008 n=5)     155.9n ± ∞ ¹   -100.00% (p=0.008 n=5)
Match/Hard/32M-4                    1380076363.0n ± ∞ ¹    1362504.0n ± ∞ ¹     -99.90% (p=0.008 n=5)     155.7n ± ∞ ¹   -100.00% (p=0.008 n=5)
Match/Hard1/16-4                          2661.0n ± ∞ ¹        697.6n ± ∞ ¹     -73.78% (p=0.008 n=5)     169.0n ± ∞ ¹    -93.65% (p=0.008 n=5)
Match/Hard1/32-4                          5084.0n ± ∞ ¹        796.9n ± ∞ ¹     -84.33% (p=0.008 n=5)     201.5n ± ∞ ¹    -96.04% (p=0.008 n=5)
Match/Hard1/1K-4                         157.089µ ± ∞ ¹        6.838µ ± ∞ ¹     -95.65% (p=0.008 n=5)     2.092µ ± ∞ ¹    -98.67% (p=0.008 n=5)
Match/Hard1/32K-4                        6748.74µ ± ∞ ¹       196.99µ ± ∞ ¹     -97.08% (p=0.008 n=5)     62.52µ ± ∞ ¹    -99.07% (p=0.008 n=5)
Match/Hard1/1M-4                         214.239m ± ∞ ¹        6.290m ± ∞ ¹     -97.06% (p=0.008 n=5)     1.895m ± ∞ ¹    -99.12% (p=0.008 n=5)
Match/Hard1/32M-4                        6936.54m ± ∞ ¹       201.05m ± ∞ ¹     -97.10% (p=0.008 n=5)     59.88m ± ∞ ¹    -99.14% (p=0.008 n=5)
MatchParallel/Easy0/16-4                   1.710n ± ∞ ¹      942.200n ± ∞ ¹  +54999.42% (p=0.008 n=5)    81.700n ± ∞ ¹  +4677.78% (p=0.008 n=5)
MatchParallel/Easy0/32-4                   19.10n ± ∞ ¹       944.80n ± ∞ ¹   +4846.60% (p=0.008 n=5)     82.03n ± ∞ ¹   +329.48% (p=0.008 n=5)
MatchParallel/Easy0/1K-4                   96.72n ± ∞ ¹       978.40n ± ∞ ¹    +911.58% (p=0.008 n=5)     80.82n ± ∞ ¹    -16.44% (p=0.008 n=5)
MatchParallel/Easy0/32K-4                1285.00n ± ∞ ¹      1425.00n ± ∞ ¹     +10.89% (p=0.008 n=5)     80.82n ± ∞ ¹    -93.71% (p=0.008 n=5)
MatchParallel/Easy0/1M-4                74284.00n ± ∞ ¹     41354.00n ± ∞ ¹     -44.33% (p=0.008 n=5)     79.92n ± ∞ ¹    -99.89% (p=0.008 n=5)
MatchParallel/Easy0/32M-4             2382580.00n ± ∞ ¹   1320813.00n ± ∞ ¹     -44.56% (p=0.008 n=5)     80.95n ± ∞ ¹   -100.00% (p=0.008 n=5)
MatchParallel/Easy0i/16-4                  1.711n ± ∞ ¹      937.600n ± ∞ ¹  +54698.36% (p=0.008 n=5)    81.650n ± ∞ ¹  +4672.06% (p=0.008 n=5)
MatchParallel/Easy0i/32-4                 339.90n ± ∞ ¹       945.70n ± ∞ ¹    +178.23% (p=0.008 n=5)     81.28n ± ∞ ¹    -76.09% (p=0.008 n=5)
MatchParallel/Easy0i/1K-4               10107.00n ± ∞ ¹       992.20n ± ∞ ¹     -90.18% (p=0.008 n=5)     81.21n ± ∞ ¹    -99.20% (p=0.008 n=5)
MatchParallel/Easy0i/32K-4             451362.00n ± ∞ ¹      1422.00n ± ∞ ¹     -99.68% (p=0.008 n=5)     81.38n ± ∞ ¹    -99.98% (p=0.008 n=5)
MatchParallel/Easy0i/1M-4            14439204.00n ± ∞ ¹     41325.00n ± ∞ ¹     -99.71% (p=0.008 n=5)     80.67n ± ∞ ¹   -100.00% (p=0.008 n=5)
MatchParallel/Easy0i/32M-4          563552821.00n ± ∞ ¹   1322234.00n ± ∞ ¹     -99.77% (p=0.008 n=5)     80.94n ± ∞ ¹   -100.00% (p=0.008 n=5)
MatchParallel/Easy1/16-4                   1.712n ± ∞ ¹      940.700n ± ∞ ¹  +54847.43% (p=0.008 n=5)    82.050n ± ∞ ¹  +4692.64% (p=0.008 n=5)
MatchParallel/Easy1/32-4                   18.04n ± ∞ ¹       944.10n ± ∞ ¹   +5133.37% (p=0.008 n=5)     81.89n ± ∞ ¹   +353.94% (p=0.008 n=5)
MatchParallel/Easy1/1K-4                  256.50n ± ∞ ¹      1007.00n ± ∞ ¹    +292.59% (p=0.008 n=5)     81.64n ± ∞ ¹    -68.17% (p=0.008 n=5)
MatchParallel/Easy1/32K-4               11781.00n ± ∞ ¹      1424.00n ± ∞ ¹     -87.91% (p=0.008 n=5)     81.45n ± ∞ ¹    -99.31% (p=0.008 n=5)
MatchParallel/Easy1/1M-4               407922.00n ± ∞ ¹     41413.00n ± ∞ ¹     -89.85% (p=0.008 n=5)     81.57n ± ∞ ¹    -99.98% (p=0.008 n=5)
MatchParallel/Easy1/32M-4            13077618.00n ± ∞ ¹   1326004.00n ± ∞ ¹     -89.86% (p=0.008 n=5)     81.48n ± ∞ ¹   -100.00% (p=0.008 n=5)
MatchParallel/Medium/16-4                  1.719n ± ∞ ¹      937.100n ± ∞ ¹  +54414.25% (p=0.008 n=5)    79.930n ± ∞ ¹  +4549.80% (p=0.008 n=5)
MatchParallel/Medium/32-4                 284.90n ± ∞ ¹       948.90n ± ∞ ¹    +233.06% (p=0.008 n=5)     79.97n ± ∞ ¹    -71.93% (p=0.008 n=5)
MatchParallel/Medium/1K-4                9223.00n ± ∞ ¹      1003.00n ± ∞ ¹     -89.13% (p=0.008 n=5)     80.01n ± ∞ ¹    -99.13% (p=0.008 n=5)
MatchParallel/Medium/32K-4             469468.00n ± ∞ ¹      1418.00n ± ∞ ¹     -99.70% (p=0.008 n=5)     79.45n ± ∞ ¹    -99.98% (p=0.008 n=5)
MatchParallel/Medium/1M-4            14414208.00n ± ∞ ¹     41372.00n ± ∞ ¹     -99.71% (p=0.008 n=5)     80.16n ± ∞ ¹   -100.00% (p=0.008 n=5)
MatchParallel/Medium/32M-4          571771553.00n ± ∞ ¹   1320563.00n ± ∞ ¹     -99.77% (p=0.008 n=5)     79.68n ± ∞ ¹   -100.00% (p=0.008 n=5)
MatchParallel/Hard/16-4                    1.717n ± ∞ ¹      940.800n ± ∞ ¹  +54693.24% (p=0.008 n=5)    81.440n ± ∞ ¹  +4643.16% (p=0.008 n=5)
MatchParallel/Hard/32-4                   487.00n ± ∞ ¹       949.10n ± ∞ ¹     +94.89% (p=0.008 n=5)     81.31n ± ∞ ¹    -83.30% (p=0.008 n=5)
MatchParallel/Hard/1K-4                 14814.00n ± ∞ ¹      1002.00n ± ∞ ¹     -93.24% (p=0.008 n=5)     81.13n ± ∞ ¹    -99.45% (p=0.008 n=5)
MatchParallel/Hard/32K-4               685271.00n ± ∞ ¹      1423.00n ± ∞ ¹     -99.79% (p=0.008 n=5)     81.37n ± ∞ ¹    -99.99% (p=0.008 n=5)
MatchParallel/Hard/1M-4              21979040.00n ± ∞ ¹     41376.00n ± ∞ ¹     -99.81% (p=0.008 n=5)     81.13n ± ∞ ¹   -100.00% (p=0.008 n=5)
MatchParallel/Hard/32M-4           1382894393.00n ± ∞ ¹   1321072.00n ± ∞ ¹     -99.90% (p=0.008 n=5)     81.32n ± ∞ ¹   -100.00% (p=0.008 n=5)
MatchParallel/Hard1/16-4                 1303.00n ± ∞ ¹      1020.00n ± ∞ ¹     -21.72% (p=0.008 n=5)     86.51n ± ∞ ¹    -93.36% (p=0.008 n=5)
MatchParallel/Hard1/32-4                 2526.00n ± ∞ ¹      1094.00n ± ∞ ¹     -56.69% (p=0.008 n=5)     97.66n ± ∞ ¹    -96.13% (p=0.008 n=5)
MatchParallel/Hard1/1K-4                 77806.0n ± ∞ ¹       3413.0n ± ∞ ¹     -95.61% (p=0.008 n=5)     633.1n ± ∞ ¹    -99.19% (p=0.008 n=5)
MatchParallel/Hard1/32K-4                3519.23µ ± ∞ ¹        97.97µ ± ∞ ¹     -97.22% (p=0.008 n=5)     17.12µ ± ∞ ¹    -99.51% (p=0.008 n=5)
MatchParallel/Hard1/1M-4                121966.5µ ± ∞ ¹       3122.6µ ± ∞ ¹     -97.44% (p=0.008 n=5)     517.1µ ± ∞ ¹    -99.58% (p=0.008 n=5)
MatchParallel/Hard1/32M-4                6725.29m ± ∞ ¹       100.80m ± ∞ ¹     -98.50% (p=0.008 n=5)     16.86m ± ∞ ¹    -99.75% (p=0.008 n=5)
```

Most benchmarks from the standard library are similar to `Find`, testing simple expressions with small input.
In all of these, the standard library performs much better. To reiterate the guidance at the top of this README,
if you only use simple expressions with small input, you should not use this library.

The compilation benchmarks show that re2 is much slower to compile expressions than the standard
library - this is more than just the overhead of foreign function invocation. This likely results
in the improved performance at runtime in other cases. Because memory allocated in Wasm or cgo is
essentially "off-heap", it is difficult to measure allocation overhead compared to standard library.
More investigation needs to be done, but we expect the memory overhead for all approaches to be
similar.

The match benchmarks show the performance tradeoffs for complexity vs input size. We see the standard
library perform the best with low complexity and size, but for high complexity or high input size,
go-re2 with WebAssembly outperforms, often significantly. Notable is `Hard1`, where even on the smallest
size this library outperforms. The expression is `ABCD|CDEF|EFGH|GHIJ|IJKL|KLMN|MNOP|OPQR|QRST|STUV|UVWX|WXYZ`,
a simple OR of literals - re2 has the concept of regex sets and likely is able to optimize this in a
special way. The CoreRuleSet contains many expressions of a form like this - this possibly indicates good
performance in real world use cases.

[1]: https://pkg.go.dev/regexp
[2]: https://github.com/google/re2
[3]: https://wazero.io
[4]: https://github.com/wasilibs/go-re2/actions/workflows/bench.yaml
[5]: https://github.com/coreruleset/coreruleset
[6]: https://github.com/corazawaf/coraza
[8]: https://www.msys2.org/
[9]: https://brew.sh/
[10]: https://github.com/corazawaf/coraza-wasilibs
