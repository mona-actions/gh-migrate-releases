# gh-migrate-releases

`gh-migrate-releases` is a [GitHub CLI](https://cli.github.com) extension to assist in the migration of releases between GitHub repositories. This extension aims to fill the gaps in the existing solutions for migrating releases. Whether you are consolidating repositories in an organization or auditing releases in an existing repository, this extension can help.

## Install

```bash
gh extension install mona-actions/gh-migrate-releases
```

## Upgrade

```bash
gh extension upgrade gh-migrate-releases
```

## Usage: Export

Creates a JSON file of the releases tied to a repository

```bash
gh migrate-releases export --hostname github.example.com -o <org-name> --repository <repo-name> --token <token>
```

```txt
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
gh migrate-releases sync --source-hostname github.example.com --source-organization <source-org> --source-token <source-token> --repository <repo-name> --target-organization <target-org> --target-token <target-token> --mapping-file "path/to/user-mappings.csv"
```

```txt
Usage:
  migrate-releases sync [flags]

Flags:
  -h, --help                          help for sync
  -m, --mapping-file string           Mapping file path to use for mapping members handles
  -r, --repository string             repository to export/import releases from/to; can't be used with --repository-list
  -l, --repository-list-file string   file path that contains list of repositories to export/import releases from/to; can't be used with --repository
  -u, --source-hostname string        GitHub Enterprise source hostname url (optional) Ex. github.example.com
  -s, --source-organization string    Source Organization to sync releases from
  -a, --source-token string           Source Organization GitHub token. Scopes: read:org, read:user, user:email
  -t, --target-organization string    Target Organization to sync releases from
  -b, --target-token string           Target Organization GitHub token. Scopes: admin:org
```

### Repository List Example

A list of repositories can be provided to sync releases from multiple repositories to many repositories in a single target.

Example:

```txt
https://github.example.com/owner/repo-name
https://github.example.com/owner/repo-name2
```

or

```txt
owner/repo-name
owner/repo-name2
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
