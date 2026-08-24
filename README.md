
# wslcd

`wslcd` resolves a target directory and prints the path to stdout.

- If given a Linux path: it behaves like `cd` (resolves `~`, relative paths, verifies directory).
- If given a Windows path (e.g., `C:\\Users\\me\\Projects`), it maps to `/mnt/c/...` and resolves path segments case-insensitively.
  - If multiple case-sensitive candidates exist, it chooses the one with the **highest overall case match score**.
  - If still tied, it sorts the full candidate paths lexicographically and picks the first.

> ⚠️ A process cannot change its parent shell's CWD. Use the shell function below so your shell performs the final `cd`.

## Install

### Debian / Ubuntu (WSL)

Download the `.deb` for your architecture from the
[latest release](https://github.com/michaelkc/wslcd/releases/latest)
(`amd64` on typical Intel/AMD machines, `arm64` on ARM), then:

```bash
sudo dpkg -i wslcd_*_amd64.deb   # or *_arm64.deb
```

This installs:

- `/usr/bin/wslcd` — the binary
- `/etc/profile.d/wslcd.sh` — a shell wrapper so `wslcd` changes directory directly

The wrapper is sourced by login shells (WSL starts login shells by default),
so after installing, open a new terminal or run `source /etc/profile` and:

```bash
wslcd C:\temp\somedir    # actually cds, no function setup needed
```

> **Fish users:** `profile.d` doesn't apply. Add to `~/.config/fish/functions/wslcd.fish`:
> ```fish
> function wslcd
>     set -l target (command wslcd $argv); and cd $target
> end
> ```

### Nix

Run without installing (any Linux system with flakes enabled):

```bash
nix run github:michaelkc/wslcd -- C:\temp\somedir
```

Or add it to your flake-based NixOS/home-manager configuration:

```nix
# flake.nix inputs
inputs.wslcd = {
  url = "github:michaelkc/wslcd";
  inputs.nixpkgs.follows = "nixpkgs"; # build with YOUR nixpkgs channel
};

# then, e.g. in configuration.nix
environment.systemPackages = [ inputs.wslcd.packages.x86_64-linux.default ];
```

On **NixOS**, the package ships `etc/profile.d/wslcd.sh`, which is sourced by
login shells automatically — `wslcd` cds out of the box.

On **home-manager standalone** or non-NixOS systems, source it yourself:

```bash
# bash: ~/.bashrc   zsh: ~/.zshrc
source "$(dirname $(readlink -f $(command -v wslcd)))/../etc/profile.d/wslcd.sh"
```

### From source

```bash
make install
# or:
make build && sudo install -m 0755 wslcd /usr/local/bin/wslcd
```

If you install from source, copy `packaging/wslcd.sh` into `/etc/profile.d/`
(or paste the shell function below into your rc file) so `wslcd` can cd.

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
source ~/.bashrc
wslcd C:\\temp\\somedir\\someotherdir
# shell will cd to /mnt/c/temp/somedir/someotherdir (best case match)
```

You can also use:
```bash
cd "$(wslcd C:\\temp\\somedir\\someotherdir)"
```

## Notes

- Windows paths may use `\\` or `/` after the drive, e.g. `C:\\Users\\me` or `C:/Users/me`.
- `..` and `.` are handled when resolving Windows paths.
- Symlinks are followed when verifying directories.
- If a path cannot be resolved to a directory, a non-zero exit code is returned with an error message on stderr.
