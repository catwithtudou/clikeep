# clikeep

clikeep is a local-first update manager for frequently used CLI tools.

## V0.1

- confirmed CLI update profiles
- dry-run update plans
- sequential update execution
- per-tool logs
- latest status
- doctor checks

## Example

```bash
clikeep init
clikeep add lark-cli --update "lark-cli update" --yes
clikeep up --dry-run
clikeep up --yes
clikeep status
clikeep doctor
```
