# gh-migrate-releases

`gh-migrate-releases` is a [GitHub CLI](https://cli.github.com) extension to assist in the migration of releases between GitHub repositories. This extension aims to fill the gaps in the existing solutions for migrating releases. Whether you are consolidating repositories in an organization or auditing releases in an existing repository, this extension can help.

## Install

```bash
gh extension install mona-actions/gh-migrate-releases
```

## Usage: Export

Creates a JSON file of the releases tied to a repository

```bash
Usage:
  migrate-releases export [flags]

Flags:
  -f, --file-prefix string    Output filenames prefix
  -h, --help                  help for export
  -u, --hostname string       GitHub Enterprise hostname url (optional) Ex. github.example.com
  -o, --organization string   Organization of the repository
  -r, --repository string     repository to export
  -t, --token string          GitHub token
```

## Usage: Sync

Recreates releases,from a source repository to a target repository

```bash
Usage:
  migrate-releases sync [flags]

Flags:
  -h, --help                         help for sync
  -m, --mapping-file string          Mapping file path to use for mapping members handles
  -r, --repository string            repository to export/import releases from/to
  -u, --source-hostname string       GitHub Enterprise source hostname url (optional) Ex. github.example.com
  -s, --source-organization string   Source Organization to sync releases from
  -a, --source-token string          Source Organization GitHub token. Scopes: read:org, read:user, user:email
  -t, --target-organization string   Target Organization to sync releases from
  -b, --target-token string          Target Organization GitHub token. Scopes: admin:org
```

### Mapping File Example

A mapping file can be provided to map member handles in case they are different between source and target.

Example:

```csv
source,target
flastname,firstname.lastname
```

## License

- [MIT](./license) (c) [Mona-Actions](https://github.com/mona-actions)
- [Contributing](./contributing.md)
