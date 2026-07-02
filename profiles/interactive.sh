# Interactive profile: relax strictness and enable session comforts.
# Safe to source from bash, zsh, or plain sh.
set +e
set +u
[ -n "${BASH_VERSION-}" ] && shopt -s histappend checkwinsize 2>/dev/null
[ -n "${ZSH_VERSION-}" ] && setopt APPEND_HISTORY NO_NOMATCH 2>/dev/null
# Sourcing must report success even when the guards above short-circuit.
true
