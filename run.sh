make build
./dist/shellenv init
./dist/shellenv create --shell bash@5.2 --profile strict
eval "$(./dist/shellenv activate)"
echo "$SHELLENV_ENV_NAME"    # -> default

# tests
make test
make itest
