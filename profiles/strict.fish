# Strict profile (fish variant)
# fish has no equivalent of `set -euo pipefail`; scripts must check $status
# (or use `; and`/`; or` chains) themselves. This variant exists so fish
# sessions still get profile sourcing — add exported variables below.
