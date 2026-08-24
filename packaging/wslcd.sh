# wslcd shell integration
#
# A process cannot change its parent shell's working directory, so this
# wrapper calls the wslcd binary and lets the shell perform the final cd.
# Sourced automatically from /etc/profile.d on Debian/Ubuntu (login shells)
# and on NixOS. For fish, see the README.

if command -v wslcd >/dev/null 2>&1; then
    wslcd() {
        local target
        # 'command' forces using the external binary, not this function
        if ! target="$(command wslcd "$@")"; then
            return 1
        fi
        [ -z "$target" ] && return 1
        cd -- "$target" || return 1
    }
fi
