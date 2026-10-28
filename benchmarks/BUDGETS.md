# Performance budgets

Budgets for `BenchmarkRenderMJML` (full pipeline: parse, render, inline CSS,
post-processing), measured on the reference development machine. Each budget is
the measured median plus ~60% headroom so normal noise does not trip it while a
structural regression does.

Policy: CI prints benchmark results but does not fail on them. Update this table
in the same commit as a change that moves a budget materially.

Measured with Go 1.24 on an AMD Ryzen 9 9950X3D, `-benchmem -count=3`, medians,
over the whole parity corpus.

| Fixture | Budget ns/op | Budget B/op | Budget allocs/op | Measured ns/op | Measured B/op | Measured allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `accordion.mjml` | 134,016 | 232,410 | 2,036 | 83,760 | 145,256 | 1,272 |
| `button-complex.mjml` | 1,073,730 | 1,686,200 | 17,204 | 671,081 | 1,053,875 | 10,752 |
| `button.mjml` | 59,767 | 100,157 | 1,093 | 37,354 | 62,598 | 683 |
| `carousel-thumbnails-hidden.mjml` | 92,498 | 227,080 | 1,604 | 57,811 | 141,925 | 1,002 |
| `carousel.mjml` | 125,458 | 327,119 | 2,146 | 78,411 | 204,449 | 1,341 |
| `component-head-styles.mjml` | 126,820 | 229,821 | 2,048 | 79,262 | 143,638 | 1,280 |
| `content.mjml` | 143,663 | 212,197 | 2,335 | 89,789 | 132,623 | 1,459 |
| `fluid-height-hero.mjml` | 79,712 | 138,160 | 1,384 | 49,820 | 86,350 | 865 |
| `head-attributes-class.mjml` | 109,664 | 175,532 | 1,935 | 68,540 | 109,707 | 1,209 |
| `head-attributes-per-tag.mjml` | 85,764 | 132,236 | 1,487 | 53,602 | 82,647 | 929 |
| `head-fonts-default.mjml` | 65,004 | 100,408 | 1,212 | 40,627 | 62,755 | 757 |
| `head-html-attributes.mjml` | 156,194 | 263,671 | 2,280 | 97,621 | 164,794 | 1,425 |
| `head-owa.mjml` | 54,026 | 87,602 | 1,032 | 33,766 | 54,751 | 645 |
| `head.mjml` | 156,325 | 256,858 | 2,095 | 97,703 | 160,536 | 1,309 |
| `image-complex.mjml` | 836,314 | 1,190,664 | 13,861 | 522,696 | 744,165 | 8,663 |
| `image.mjml` | 59,810 | 100,391 | 1,151 | 37,381 | 62,744 | 719 |
| `input_big.mjml` | 4,277,055 | 6,158,615 | 52,706 | 2,673,159 | 3,849,134 | 32,941 |
| `layout.mjml` | 265,868 | 427,703 | 4,772 | 166,167 | 267,314 | 2,982 |
| `navbar.mjml` | 88,293 | 152,002 | 1,485 | 55,183 | 95,001 | 928 |
| `raw-in-column.mjml` | 55,476 | 90,426 | 1,064 | 34,672 | 56,516 | 665 |
| `raw-in-group.mjml` | 90,608 | 139,476 | 1,666 | 56,630 | 87,172 | 1,041 |
| `raw-in-section.mjml` | 77,058 | 123,463 | 1,453 | 48,161 | 77,164 | 908 |
| `raw-position-file-start.mjml` | 52,935 | 91,028 | 1,021 | 33,084 | 56,892 | 638 |
| `raw.mjml` | 56,652 | 89,997 | 1,063 | 35,407 | 56,248 | 664 |
| `section-container-width.mjml` | 100,311 | 152,207 | 1,837 | 62,694 | 95,129 | 1,148 |
| `simple-hero.mjml` | 78,392 | 136,727 | 1,392 | 48,995 | 85,454 | 870 |
| `simple.mjml` | 24,106 | 42,128 | 506 | 15,066 | 26,330 | 316 |
| `social.mjml` | 154,301 | 296,376 | 2,474 | 96,438 | 185,235 | 1,546 |
| `table-width-auto.mjml` | 55,831 | 91,055 | 1,042 | 34,894 | 56,909 | 651 |
| `upstream/mjml--accordion-fontFamily--00.mjml` | 135,269 | 246,045 | 2,250 | 84,543 | 153,778 | 1,406 |
| `upstream/mjml--accordion-padding--00.mjml` | 86,152 | 153,250 | 1,397 | 53,845 | 95,781 | 873 |
| `upstream/mjml--accordionTitle-fontWeight--00.mjml` | 134,608 | 224,589 | 1,898 | 84,130 | 140,368 | 1,186 |
| `upstream/mjml--beautify-output--00.mjml` | 51,631 | 84,469 | 996 | 32,269 | 52,793 | 622 |
| `upstream/mjml--beautify-output--01.mjml` | 51,935 | 84,631 | 997 | 32,459 | 52,894 | 623 |
| `upstream/mjml--beautify-output--02.mjml` | 43,844 | 69,829 | 840 | 27,402 | 43,643 | 525 |
| `upstream/mjml--beautify-output--03.mjml` | 51,660 | 84,456 | 996 | 32,287 | 52,785 | 622 |
| `upstream/mjml--beautify-output--04.mjml` | 52,348 | 84,712 | 997 | 32,717 | 52,945 | 623 |
| `upstream/mjml--carousel-hoverSupported--00.mjml` | 127,101 | 322,682 | 2,052 | 79,438 | 201,676 | 1,282 |
| `upstream/mjml--column-border-radius--00.mjml` | 60,986 | 101,666 | 1,130 | 38,116 | 63,541 | 706 |
| `upstream/mjml--html-attributes--00.mjml` | 178,626 | 300,760 | 2,503 | 111,641 | 187,975 | 1,564 |
| `upstream/mjml--html-comments--00.mjml` | 54,148 | 88,692 | 1,000 | 33,842 | 55,432 | 625 |
| `upstream/mjml--htmlmin-ignore--00.mjml` | 44,100 | 77,183 | 842 | 27,562 | 48,239 | 526 |
| `upstream/mjml--htmlmin-ignore--01.mjml` | 45,087 | 71,876 | 842 | 28,179 | 44,922 | 526 |
| `upstream/mjml--htmlmin-ignore--02.mjml` | 43,276 | 69,884 | 840 | 27,047 | 43,677 | 525 |
| `upstream/mjml--navbar-ico-padding--00.mjml` | 86,861 | 153,407 | 1,460 | 54,288 | 95,879 | 912 |
| `upstream/mjml--social-align--00.mjml` | 87,933 | 155,562 | 1,500 | 54,958 | 97,226 | 937 |
| `upstream/mjml--social-icon-height--00.mjml` | 93,229 | 166,496 | 1,559 | 58,268 | 104,060 | 974 |
| `upstream/mjml--table-cellspacing--00.mjml` | 62,239 | 104,295 | 1,071 | 38,899 | 65,184 | 669 |
| `upstream/mjml--tableWidth--00.mjml` | 205,386 | 326,927 | 3,388 | 128,366 | 204,329 | 2,117 |
| `upstream/mjml--template-syntax-sanitization--00.mjml` | 42,605 | 69,930 | 840 | 26,628 | 43,706 | 525 |
| `upstream/mjml--template-syntax-sanitization--01.mjml` | 52,871 | 88,085 | 1,031 | 33,044 | 55,053 | 644 |
| `upstream/mjml--template-syntax-sanitization--02.mjml` | 41,796 | 69,100 | 839 | 26,122 | 43,187 | 524 |
| `upstream/mjml--template-syntax-sanitization--03.mjml` | 41,722 | 69,122 | 839 | 26,076 | 43,201 | 524 |
| `upstream/mjml--template-syntax-sanitization--04.mjml` | 45,066 | 71,508 | 880 | 28,166 | 44,692 | 550 |
| `upstream/mjml--template-syntax-sanitization--05.mjml` | 41,924 | 69,111 | 839 | 26,202 | 43,194 | 524 |
| `upstream/mjml--template-syntax-sanitization--06.mjml` | 42,172 | 69,311 | 839 | 26,357 | 43,319 | 524 |
| `upstream/mjml--template-syntax-sanitization--07.mjml` | 42,359 | 69,242 | 839 | 26,474 | 43,276 | 524 |
| `upstream/mjml--wrapper-border-radius--00.mjml` | 64,023 | 102,655 | 1,242 | 40,014 | 64,159 | 776 |
| `upstream/mjml--wrapper-gap--00.mjml` | 148,543 | 245,661 | 2,736 | 92,839 | 153,538 | 1,710 |
| `upstream/parser-xml--preprocessors--00.mjml` | 51,290 | 84,778 | 1,004 | 32,056 | 52,986 | 627 |

`BenchmarkParse` and `BenchmarkParityComparison` isolate the parser and the test
harness; inline-CSS cost is included in the end-to-end numbers until the inliner
rewrite (M3) exposes a separate seam.

## Reproducing

```bash
go test ./ -run XXX -bench BenchmarkRenderMJML -benchmem -count=3
go test ./ -run XXX -bench BenchmarkParse -benchmem
go test ./ -run XXX -bench BenchmarkParityComparison -benchmem
```
