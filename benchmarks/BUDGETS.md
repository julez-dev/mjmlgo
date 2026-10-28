# Performance budgets

Budgets for `BenchmarkRenderMJML` (full pipeline: parse, validate, render, inline
CSS, post-processing), measured on the reference development machine. Each budget
is the measured median plus ~60% headroom so normal noise does not trip it while a
structural regression does.

Policy: CI prints benchmark results but does not fail on them. Update this table
in the same commit as a change that moves a budget materially.

Measured with Go 1.24 on an AMD Ryzen 9 9950X3D, `-benchmem -count=3`, medians,
over the whole parity corpus. A fixture with a `.cli-args` sidecar is measured
with the options it records, from memory; fixtures with `mj-include` are measured
through `Renderer.RenderString` (fs reads included).

| Fixture | Budget ns/op | Budget B/op | Budget allocs/op | Measured ns/op | Measured B/op | Measured allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `accordion.mjml` | 139,018 | 265,565 | 2,514 | 86,886 | 165,978 | 1,571 |
| `button-complex.mjml` | 1,034,053 | 1,971,773 | 20,861 | 646,283 | 1,232,358 | 13,038 |
| `button.mjml` | 57,855 | 113,272 | 1,266 | 36,159 | 70,795 | 791 |
| `carousel-thumbnails-hidden.mjml` | 93,077 | 242,060 | 1,818 | 58,173 | 151,287 | 1,136 |
| `carousel.mjml` | 128,741 | 345,640 | 2,418 | 80,463 | 216,025 | 1,511 |
| `comments-between-components.mjml` | 124,325 | 257,898 | 2,838 | 77,703 | 161,186 | 1,774 |
| `comments-containers.mjml` | 194,027 | 365,347 | 3,488 | 121,267 | 228,342 | 2,180 |
| `comments-head.mjml` | 58,405 | 109,282 | 1,299 | 36,503 | 68,301 | 812 |
| `comments-keep-false.mjml` | 63,245 | 110,979 | 1,242 | 39,528 | 69,362 | 776 |
| `comments-raw.mjml` | 57,366 | 116,206 | 1,246 | 35,854 | 72,629 | 779 |
| `component-head-styles.mjml` | 124,703 | 258,346 | 2,440 | 77,939 | 161,466 | 1,525 |
| `content.mjml` | 143,872 | 243,967 | 2,773 | 89,920 | 152,479 | 1,733 |
| `fluid-height-hero.mjml` | 79,394 | 154,456 | 1,600 | 49,621 | 96,535 | 1,000 |
| `head-attributes-class.mjml` | 113,821 | 210,189 | 2,420 | 71,138 | 131,368 | 1,512 |
| `head-attributes-per-tag.mjml` | 87,988 | 162,832 | 1,892 | 54,992 | 101,770 | 1,182 |
| `head-fonts-custom.mjml` | 81,483 | 148,102 | 1,675 | 50,927 | 92,564 | 1,047 |
| `head-fonts-default.mjml` | 62,144 | 115,106 | 1,415 | 38,840 | 71,941 | 884 |
| `head-html-attributes.mjml` | 144,826 | 228,724 | 2,540 | 90,516 | 142,952 | 1,587 |
| `head-owa.mjml` | 52,551 | 98,589 | 1,186 | 32,844 | 61,618 | 741 |
| `head.mjml` | 146,082 | 223,896 | 2,378 | 91,301 | 139,935 | 1,486 |
| `ignore-includes/main.mjml` | 57,562 | 105,930 | 1,240 | 35,976 | 66,206 | 775 |
| `image-complex.mjml` | 800,258 | 1,395,453 | 16,647 | 500,161 | 872,158 | 10,404 |
| `image.mjml` | 61,725 | 112,573 | 1,330 | 38,578 | 70,358 | 831 |
| `includes/main.mjml` | 449,231 | 669,952 | 6,360 | 280,769 | 418,720 | 3,975 |
| `input_big.mjml` | 4,115,906 | 5,475,077 | 57,148 | 2,572,441 | 3,421,923 | 35,717 |
| `layout.mjml` | 270,480 | 500,234 | 5,765 | 169,050 | 312,646 | 3,603 |
| `navbar.mjml` | 91,592 | 172,783 | 1,783 | 57,245 | 107,989 | 1,114 |
| `printer-support.mjml` | 89,856 | 152,194 | 1,773 | 56,160 | 95,121 | 1,108 |
| `raw-in-column.mjml` | 58,983 | 116,711 | 1,244 | 36,864 | 72,944 | 777 |
| `raw-in-group.mjml` | 86,856 | 161,021 | 1,960 | 54,285 | 100,638 | 1,225 |
| `raw-in-section.mjml` | 76,111 | 143,559 | 1,733 | 47,569 | 89,724 | 1,083 |
| `raw-position-file-start.mjml` | 52,802 | 102,541 | 1,181 | 33,001 | 64,088 | 738 |
| `raw.mjml` | 57,752 | 116,128 | 1,240 | 36,095 | 72,580 | 775 |
| `section-container-width.mjml` | 100,378 | 175,112 | 2,172 | 62,736 | 109,445 | 1,357 |
| `simple-hero.mjml` | 81,484 | 153,000 | 1,618 | 50,927 | 95,625 | 1,011 |
| `simple.mjml` | 23,637 | 46,032 | 562 | 14,773 | 28,770 | 351 |
| `social.mjml` | 153,253 | 326,072 | 2,861 | 95,783 | 203,795 | 1,788 |
| `table-width-auto.mjml` | 54,404 | 101,888 | 1,194 | 34,002 | 63,680 | 746 |
| `upstream/mjml--accordion-fontFamily--00.mjml` | 138,970 | 280,362 | 2,716 | 86,856 | 175,226 | 1,697 |
| `upstream/mjml--accordion-padding--00.mjml` | 89,760 | 174,172 | 1,709 | 56,100 | 108,857 | 1,068 |
| `upstream/mjml--accordionTitle-fontWeight--00.mjml` | 147,098 | 262,418 | 2,476 | 91,936 | 164,011 | 1,547 |
| `upstream/mjml--beautify-output--00.mjml` | 50,741 | 95,268 | 1,149 | 31,713 | 59,542 | 718 |
| `upstream/mjml--beautify-output--01.mjml` | 51,192 | 95,492 | 1,151 | 31,995 | 59,682 | 719 |
| `upstream/mjml--beautify-output--02.mjml` | 46,565 | 88,605 | 968 | 29,103 | 55,378 | 605 |
| `upstream/mjml--beautify-output--03.mjml` | 50,474 | 95,234 | 1,149 | 31,546 | 59,521 | 718 |
| `upstream/mjml--beautify-output--04.mjml` | 51,061 | 95,540 | 1,151 | 31,913 | 59,712 | 719 |
| `upstream/mjml--carousel-hoverSupported--00.mjml` | 123,748 | 338,962 | 2,296 | 77,342 | 211,851 | 1,435 |
| `upstream/mjml--column-border-radius--00.mjml` | 61,324 | 113,647 | 1,298 | 38,327 | 71,029 | 811 |
| `upstream/mjml--html-attributes--00.mjml` | 163,605 | 274,410 | 2,799 | 102,253 | 171,506 | 1,749 |
| `upstream/mjml--html-comments--00.mjml` | 53,453 | 99,583 | 1,154 | 33,408 | 62,239 | 721 |
| `upstream/mjml--htmlmin-ignore--00.mjml` | 47,400 | 95,671 | 970 | 29,625 | 59,794 | 606 |
| `upstream/mjml--htmlmin-ignore--01.mjml` | 48,192 | 91,176 | 970 | 30,120 | 56,985 | 606 |
| `upstream/mjml--htmlmin-ignore--02.mjml` | 46,391 | 88,669 | 968 | 28,994 | 55,418 | 605 |
| `upstream/mjml--navbar-ico-padding--00.mjml` | 88,207 | 172,807 | 1,738 | 55,129 | 108,004 | 1,086 |
| `upstream/mjml--social-align--00.mjml` | 86,628 | 172,954 | 1,732 | 54,142 | 108,096 | 1,082 |
| `upstream/mjml--social-icon-height--00.mjml` | 92,170 | 184,322 | 1,799 | 57,606 | 115,201 | 1,124 |
| `upstream/mjml--table-cellspacing--00.mjml` | 63,095 | 115,068 | 1,223 | 39,434 | 71,917 | 764 |
| `upstream/mjml--tableWidth--00.mjml` | 202,461 | 372,546 | 4,012 | 126,538 | 232,841 | 2,507 |
| `upstream/mjml--template-syntax-sanitization--00.mjml` | 46,525 | 88,722 | 968 | 29,078 | 55,451 | 605 |
| `upstream/mjml--template-syntax-sanitization--01.mjml` | 53,069 | 99,624 | 1,192 | 33,168 | 62,265 | 745 |
| `upstream/mjml--template-syntax-sanitization--02.mjml` | 45,538 | 87,951 | 967 | 28,461 | 54,969 | 604 |
| `upstream/mjml--template-syntax-sanitization--03.mjml` | 45,847 | 87,796 | 967 | 28,654 | 54,872 | 604 |
| `upstream/mjml--template-syntax-sanitization--04.mjml` | 49,016 | 90,711 | 1,016 | 30,635 | 56,694 | 635 |
| `upstream/mjml--template-syntax-sanitization--05.mjml` | 45,500 | 87,944 | 967 | 28,437 | 54,965 | 604 |
| `upstream/mjml--template-syntax-sanitization--06.mjml` | 45,864 | 88,116 | 967 | 28,665 | 55,072 | 604 |
| `upstream/mjml--template-syntax-sanitization--07.mjml` | 45,925 | 88,029 | 967 | 28,703 | 55,018 | 604 |
| `upstream/mjml--wrapper-border-radius--00.mjml` | 65,048 | 117,445 | 1,453 | 40,655 | 73,403 | 908 |
| `upstream/mjml--wrapper-gap--00.mjml` | 146,074 | 282,128 | 3,253 | 91,296 | 176,330 | 2,033 |
| `upstream/parser-xml--preprocessors--00.mjml` | 54,172 | 98,869 | 1,220 | 33,857 | 61,793 | 762 |

## Known follow-ups

- The inline stage is dominated by the post-render `html.Parse`/`html.Render`
  round-trip (~0.54 ms of `input_big`, ~65% of the inline stage). Removing it is a
  tracked structural lane.
- Validation allocates per authored attribute (`validateUnitValues`/`validateType`)
  and every component builds its `AllowedAttributes()` map per instance; hoisting
  those tables to package level recovered ~13k allocations per `input_big` render.

`BenchmarkParse` and `BenchmarkParityComparison` isolate the parser and the test
harness.

## Reproducing

```bash
go test ./ -run XXX -bench BenchmarkRenderMJML -benchmem -count=3
go test ./ -run XXX -bench BenchmarkParse -benchmem
go test ./ -run XXX -bench BenchmarkParityComparison -benchmem
```
