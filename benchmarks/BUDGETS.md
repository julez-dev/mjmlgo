# Performance budgets

Budgets for `BenchmarkRenderMJML` (full pipeline: parse, render, inline CSS,
post-processing), measured on the reference development machine. Each budget is
the measured median plus ~60% headroom so normal noise does not trip it, while a
structural regression does.

Policy: CI prints the benchmark comparison but does **not** fail on it. Update this
table (and the numbers) whenever a change moves a budget materially, in the same
commit as that change.

Measured with Go 1.24 on an AMD Ryzen 9 9950X3D, `-benchmem -count=3`, medians.
Fixtures that do not render yet (tracked by the parity baseline) are skipped.

| Fixture | Budget ns/op | Budget B/op | Budget allocs/op | Measured ns/op | Measured B/op | Measured allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `button-complex.mjml` | 1,644,887 | 2,601,906 | 20,514 | 1,028,054 | 1,626,191 | 12,821 |
| `button.mjml` | 98,328 | 164,364 | 1,360 | 61,455 | 102,727 | 850 |
| `content.mjml` | 220,092 | 320,204 | 2,928 | 137,557 | 200,127 | 1,830 |
| `fluid-height-hero.mjml` | 119,416 | 198,320 | 1,692 | 74,635 | 123,950 | 1,057 |
| `head.mjml` | 131,981 | 202,210 | 1,951 | 82,488 | 126,381 | 1,219 |
| `image-complex.mjml` | 1,173,826 | 1,883,031 | 16,464 | 733,641 | 1,176,894 | 10,290 |
| `image.mjml` | 92,943 | 161,628 | 1,421 | 58,089 | 101,017 | 888 |
| `input_big.mjml` | 4,259,069 | 5,974,176 | 54,679 | 2,661,918 | 3,733,860 | 34,174 |
| `layout.mjml` | 394,327 | 646,154 | 5,525 | 246,454 | 403,846 | 3,453 |
| `simple-hero.mjml` | 112,448 | 197,463 | 1,690 | 70,280 | 123,414 | 1,056 |
| `simple.mjml` | 41,349 | 79,530 | 668 | 25,843 | 49,706 | 417 |
| `social.mjml` | 222,404 | 440,727 | 3,012 | 139,002 | 275,454 | 1,882 |
| `upstream/mjml--beautify-output--00.mjml` | 75,412 | 136,736 | 1,167 | 47,132 | 85,460 | 729 |
| `upstream/mjml--beautify-output--01.mjml` | 75,951 | 137,008 | 1,167 | 47,469 | 85,630 | 729 |
| `upstream/mjml--beautify-output--03.mjml` | 75,162 | 136,648 | 1,167 | 46,976 | 85,405 | 729 |
| `upstream/mjml--beautify-output--04.mjml` | 76,506 | 137,058 | 1,167 | 47,816 | 85,661 | 729 |
| `upstream/mjml--column-border-radius--00.mjml` | 86,236 | 155,434 | 1,304 | 53,897 | 97,146 | 815 |
| `upstream/mjml--html-comments--00.mjml` | 83,504 | 151,375 | 1,269 | 52,190 | 94,609 | 793 |
| `upstream/mjml--social-align--00.mjml` | 122,452 | 217,754 | 1,773 | 76,532 | 136,096 | 1,108 |
| `upstream/mjml--social-icon-height--00.mjml` | 134,292 | 256,642 | 1,858 | 83,932 | 160,401 | 1,161 |
| `upstream/mjml--wrapper-border-radius--00.mjml` | 103,421 | 176,018 | 1,564 | 64,638 | 110,011 | 977 |
| `upstream/mjml--wrapper-gap--00.mjml` | 225,757 | 401,495 | 3,359 | 141,098 | 250,934 | 2,099 |
| `upstream/parser-xml--preprocessors--00.mjml` | 76,122 | 136,901 | 1,175 | 47,576 | 85,563 | 734 |

Stage-level budgets are tracked from profiles for now; `BenchmarkParse` and
`BenchmarkParityComparison` isolate the parser and the test harness. Inline CSS
inlining is measured through the end-to-end benchmark until the inliner rewrite
(M3) exposes a separate seam.

## Reproducing

```bash
go test ./ -run XXX -bench BenchmarkRenderMJML -benchmem -count=3
go test ./ -run XXX -bench BenchmarkParse -benchmem
go test ./ -run XXX -bench BenchmarkParityComparison -benchmem
```
