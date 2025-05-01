#!/usr/bin/env bash
set -euo pipefail

# # https://stackoverflow.com/a/78585264
# export CGO_LDFLAGS="-Wl,-no_warn_duplicate_libraries"
# export CGO_ENABLED=0

my_absolute_dir="$(dirname -- "$(realpath "${BASH_SOURCE[0]}")")"

function truncate_logs() {
	# Default to 1000 lines if not specified
	MAX_LINES=${1:-1000}
	shift # Remove max_lines argument

	# Check for -- separator
	if [ "$1" != "--" ]; then
		echo "Error: Missing -- separator after max_lines"
		exit 1
	fi
	shift # Remove -- separator

	# Run the command and pipe through head
	"$@" | {
		# Use head to limit output, but always show test summary at the end
		head -n "$MAX_LINES"

		# If there was more output, show a message
		if [ -n "$(cat)" ]; then
			echo "... [Output truncated after $MAX_LINES lines. Set MAX_LINES=all to see full output] ..."
		fi
	}

	# Preserve the exit code of the original command
	exit "${PIPESTATUS[0]}"
}

# if mod tidy, run the task mod-tidy
if [ "${1:-}" == "mod" ] && [ "${2:-}" == "tidy" ]; then
	./task go-mod-tidy
	exit $?
fi

# if mod upgrade, run the task mod-upgrade
if [ "${1:-}" == "mod" ] && [ "${2:-}" == "upgrade" ]; then
	./task go-mod-upgrade
	exit $?
fi

test_runtime_keys=("-run")

# if first argument is "test", use gotestsum
if [ "${1:-}" == "test" ]; then
	shift

	cc=0
	ff=0
	codesign=0
	real_args=()
	runtime_args=()
	ide=0
	vv=0
	extra_args=""
	max_lines=1000 # Default to 1000 lines
	target_dir=""
	next_is_runtime_arg_key=""

	# Handle each argument
	for arg in "$@"; do
		if [ -n "$next_is_runtime_arg_key" ]; then
			runtime_args+=("${next_is_runtime_arg_key}='${arg}'")
			next_is_runtime_arg_key=""
		elif [ "$arg" = "-function-coverage" ]; then
			cc=1
		elif [ "$arg" = "-force" ]; then
			ff=1
		elif [ "$arg" = "-codesign" ]; then
			codesign=1
		elif [ "$arg" = "-v" ]; then
			vv=1
			real_args+=("$arg")
		elif [ "$arg" = "-ide" ]; then
			ide=1
		elif [[ "$arg" =~ ^-max-lines=(.*)$ ]]; then
			max_lines="${BASH_REMATCH[1]}"
		elif [[ "$arg" =~ ^./ || "$arg" =~ ^github\.com ]]; then
			target_dir="$arg"
		else
			ok=0
			for key in "${test_runtime_keys[@]}"; do
				if [[ "$arg" =~ ^$key=(.*)$ ]]; then
					runtime_args+=("$arg")
					ok=1
					break
				elif [[ "$arg" =~ ^$key$ ]]; then
					next_is_runtime_arg_key="$key"
					ok=1
					break
				fi
			done
			if [ "$ok" -eq 0 ]; then
				real_args+=("$arg")
			fi
		fi
	done

	adjusted_runtime_args=()
	for arg in "${runtime_args[@]}"; do
		# replace -xxx with -test.xxx
		adjusted_runtime_args+=("${arg//-/-test.}")
	done

	if [[ "$cc" == "1" ]]; then
		tmpcoverdir=$(mktemp -d)
		function print_coverage() {
			echo "================================================"
			echo "Function Coverage"
			echo "------------------------------------------------"
			go tool cover -func=$tmpcoverdir/coverage.out
			echo "================================================"

		}
		extra_args=" -coverprofile=$tmpcoverdir/coverage.out -covermode=atomic "
		trap "print_coverage" EXIT
	fi

	if [[ "$ff" == "1" ]]; then
		extra_args="$extra_args -count=1 "
	fi

	# grab the packages in the target directory

	raw_args=""

	fmt="pkgname"
	fmt_icon="hivis"

	# build the test binary
	if [[ "$codesign" == "1" ]]; then
		# items=$(go list -f '{{.ImportPath}}' "$target_dir")
		# project_root_dir="$(dirname -- "${BASH_SOURCE[0]}")"

		# entitlements_file="$project_root_dir/entitlements.plist"
		# if [[ ! -f "$entitlements_file" ]]; then
		# 	echo "Error: entitlements.plist file not found in project root - it is required for codesigning"
		# 	exit 1
		# fi

		# tmpdir=$(mktemp -d)
		# remove_tmpdir() {
		# 	rm -rf "$tmpdir"
		# }
		# trap remove_tmpdir EXIT
		# for item in $items; do
		# 	module=$item
		# 	file_name=$(basename "$module")
		# 	raw_args+="go test -c -o $tmpdir -v -vet=all -json -cover $extra_args ${real_args[*]} $module"
		# 	raw_args+=" && [ -f \"$tmpdir/$file_name.test\" ] "
		# 	raw_args+=" && codesign --entitlements $entitlements_file --verbose=0 -s - $tmpdir/$file_name.test "
		# 	raw_args+=" && go tool test2json -t -p $module -- $tmpdir/$file_name.test -test.v ${adjusted_runtime_args[*]} || true; "
		# done

		extra_args+="-exec 'go run $my_absolute_dir/cmd/codesign'"
	fi
	raw_args="go test -v -vet=all -json -cover $extra_args ${real_args[*]} $target_dir"
	if [[ "$vv" == "1" ]]; then
		echo "calling: $raw_args"
	fi
	if [[ "$ide" == "1" ]]; then
		bash -c "$raw_args"
	else
		truncate_logs "$max_lines" -- go tool gotest.tools/gotestsum \
			--format "$fmt" \
			--format-icons "$fmt_icon" \
			--raw-command -- bash -c "$raw_args"
	fi

	exit $?
fi

if [ "${1:-}" == "tool" ]; then
	shift
	escape_regex() {
		printf '%s\n' "$1" | sed 's/[&/\]/\\&/g'
	}
	errors_to_suppress=(
		# https://github.com/protocolbuffers/protobuf-javascript/issues/148
		"plugin.proto#L122"
		"# github.com/lima-vm/lima/cmd/limactl"
		"ld: warning: ignoring duplicate libraries: '-lobjc'"
	)
	# 🔧 Build regex for suppressing errors
	errors_to_suppress_regex=""
	for phrase in "${errors_to_suppress[@]}"; do
		escaped_phrase=$(escape_regex "$phrase")
		if [[ -n "$errors_to_suppress_regex" ]]; then
			errors_to_suppress_regex+="|"
		fi
		errors_to_suppress_regex+="$escaped_phrase"
	done

	# 'go tool -n "$@"' can but used to get the binary name that is being run in case we need it later
	# tool_binary_executable=$(go tool -n "$@")

	stdouts_to_suppress=(
		# "# github.com/lima-vm/lima/cmd/limactl"
		"invalid string just to have something heree"
		# "ld: warning: ignoring duplicate libraries: '-lobjc'"
	)
	# 🔧 Build regex for suppressing stdouts
	stdouts_to_suppress_regex=""
	for phrase in "${stdouts_to_suppress[@]}"; do
		escaped_phrase=$(escape_regex "$phrase")
		if [[ -n "$stdouts_to_suppress_regex" ]]; then
			stdouts_to_suppress_regex+="|"
		fi
		stdouts_to_suppress_regex+="$escaped_phrase"
	done

	export HL_CONFIG=./hl-config.yaml

	go tool "$@"

	exit $?
fi

# otherwise run go directly with all arguments
go "$@"
