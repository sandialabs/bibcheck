# Contributing

## Running Tests

Run tests with a shirty key or an openrouter key

```bash
go test -v ./... -args --shirty-api-key="sk-..."
go test -v ./... -args --openrouter-api-key="sk-or-v1-..."
```

## Release Deployments

* Create fine-grained token with "Contents" repository permissions (write)
* Add it as an actions secret: `RELEASE_TOKEN`

## Licensing

To list third-party licenses:

save the following as `template.md`
```md
{{ range . }}
## {{ .Name }}

* Name: {{ .Name }}
* Version: {{ .Version }}
* License: [{{ .LicenseName }}]({{ .LicenseURL }})

```
{{ .LicenseText }}
```
{{ end }}
```

```bash
go install github.com/google/go-licenses/v2@latest

go-licenses report ./... --ignore github.com/sandialabs/bibcheck --template template.md > NOTICE
```