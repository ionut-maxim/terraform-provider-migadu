# terraform-provider-migadu

A Terraform provider for [Migadu](https://migadu.com).

## Requirements

- Terraform >= 1.0
- Go >= 1.26 (to build from source)

## Usage

```hcl
provider "migadu" {
  username = "user@example.com"
  api_key  = "your-api-key"
}
```

The provider manages the following resources:

- `migadu_domain`
- `migadu_mailbox`
- `migadu_alias`
- `migadu_rewrite`
- `migadu_identity`
- `migadu_forwarding`

Each resource also has a corresponding data source.

## Development

Build:

```sh
mise run build
```

Run tests (creates real Migadu resources, needs valid credentials):

```sh
mise run testacc
```

Regenerate docs:

```sh
mise run generate
```
