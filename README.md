
# wslcd

`wslcd` resolves a target directory and prints the path to stdout.

- If given a Linux path: it behaves like `cd` (resolves `~`, relative paths, verifies directory).
- If given a Windows path (e.g., `C:\\Users\\me\\Projects`), it maps to `/mnt/c/...` and resolves path segments case-insensitively.
  - If multiple case-sensitive candidates exist, it chooses the one with the **highest overall case match score**.
  - If still tied, it sorts the full candidate paths lexicographically and picks the first.

> ⚠️ A process cannot change its parent shell's CWD. Use the shell function below so your shell performs the final `cd`.

## Install

```bash
make install
# or:
make build && sudo install -m 0755 wslcd /usr/local/bin/wslcd
```

## Usage

**Direct (prints the resolved path):**
```bash
wslcd C:\\Temp\\MyDir
# -> /mnt/c/Temp/MyDir
```

**Recommended shell function (Bash/Zsh) to actually change directory:**
```bash
wslcd() {
  local target
  # 'command' forces using the external binary, not this function
  if ! target="$(command wslcd "$@")"; then
    return 1
  fi
  [ -z "$target" ] && return 1
  cd -- "$target"
}
```

Now:
```bash
wslcd C:\\junk\\somedir\\someotherdir
# shell will cd to /mnt/c/junk/somedir/someotherdir (best case match)
```

You can also use:
```bash
cd "$(wslcd C:\\junk\\somedir\\someotherdir)"
```

## Notes

- Windows paths may use `\\` or `/` after the drive, e.g. `C:\\Users\\me` or `C:/Users/me`.
- `..` and `.` are handled when resolving Windows paths.
- Symlinks are followed when verifying directories.
- If a path cannot be resolved to a directory, a non-zero exit code is returned with an error message on stderr.
