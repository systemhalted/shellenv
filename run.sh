make build
./dist/shellenv init
./dist/shellenv create --shell bash@5.2 --profile strict
# exec/activate warn if bash@5.2 isn't installed and fall back to the system
# shell; run `./dist/shellenv install bash@5.2` (source build, minutes) to pin.
eval "$(./dist/shellenv activate)"
echo "$SHELLENV_ENV_NAME"    # -> default

# tests
make test
make itest
