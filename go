#!/usr/bin/env bash
set -euo pipefail

# # https://stackoverflow.com/a/78585264
# export CGO_LDFLAGS="-Wl,-no_warn_duplicate_libraries"
# export CGO_ENABLED=0

# echo "REENTERRRRR"

my_absolute_dir="$(dirname -- "$(realpath "${BASH_SOURCE[0]}")")"
touch "$my_absolute_dir/enter_with_call.log"
echo "ENTER WITH CALL: (cd $(pwd) && $0 $*)" >> "$my_absolute_dir/enter_with_call.log"

function safe_go_path() {
	local reset=0
	if [[ "$PATH" = *"$my_absolute_dir"* ]]; then
		export PATH=$(echo "$PATH" | sed "s|$my_absolute_dir:||")
		reset=1
	fi
	safe_go_path=$(which go)
	if [[ "$reset" == "1" ]]; then
		export PATH="$my_absolute_dir:$PATH"
	fi

	echo "$safe_go_path"

}

function safe_go() {
	local safe_go_abs_path
	safe_go_abs_path=$(safe_go_path)
	echo "safe_go: $safe_go_abs_path $*" >> "$my_absolute_dir/enter_with_call.log"
	$safe_go_abs_path "$@"
	# carry the exit code
	return "$?"
}

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
test_build_keys=("-gcflags" "-o")

real_go_binary=$(safe_go_path)

if [ "${1:-}" == "dap" ]; then
	shift
	export PATH="$my_absolute_dir:$PATH"
	dlv dap "$@"
	exit $?
fi

# if first argument is "test", use gotestsum
if [ "${1:-}" == "test" ]; then
	shift

	cc=0
	ff=0
	codesign=0
	real_args=()
	runtime_args=()
	build_args=()
	ide=0
	vv=0
	extra_args=""
	max_lines=1000 # Default to 1000 lines
	target_dir=""
	next_is_runtime_arg_key=""
	next_is_build_arg_key=""
	debug=0
	is_dap=0
	output_file=""
	gcflags_arg=""

	# Handle each argument
	for arg in "$@"; do
		if [ -n "$next_is_runtime_arg_key" ]; then
			runtime_args+=("${next_is_runtime_arg_key}='${arg}'")
			next_is_runtime_arg_key=""
		elif [ -n "$next_is_build_arg_key" ]; then
			build_args+=("${next_is_build_arg_key}='${arg}'")
			if [[ "$next_is_build_arg_key" = "-o" ]]; then
				# if arg has a __debug in it then its a is_dap
				if [[ "$arg" = *"__debug"* ]]; then
					is_dap=1
				fi
				output_file="$arg"
			elif [[ "$next_is_build_arg_key" = "-gcflags" ]]; then
				gcflags_arg="$arg"
			fi
			next_is_build_arg_key=""
		elif [ "$arg" = "-function-coverage" ]; then
			cc=1
		elif [ "$arg" = "-force" ]; then
			ff=1
		elif [ "$arg" = "-codesign" ]; then
			codesign=1
		elif [ "$arg" = "-debug" ]; then
			debug=1
		elif [ "$arg" = "-v" ]; then
			vv=1
			real_args+=("$arg")
		elif [ "$arg" = "-ide" ]; then
			ide=1
		elif [[ "$arg" =~ ^-max-lines=(.*)$ ]]; then
			max_lines="${BASH_REMATCH[1]}"
		elif [[ "$arg" =~ ^./ || "$arg" =~ ^github\.com || "$arg" = '.' ]]; then
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
			for key in "${test_build_keys[@]}"; do
				if [[ "$arg" =~ ^$key=(.*)$ ]]; then
					build_args+=("$arg")
					# if the key starts with -gcflags then we need to set the gcflags_arg
					if [[ "$key" = "-gcflags" ]]; then
						arg_no_key=$(echo "$arg" | sed "s|$key=||")
						gcflags_arg="$arg_no_key"
					fi
					ok=1
					break
				elif [[ "$arg" =~ ^$key$ ]]; then
					next_is_build_arg_key="$key"
					ok=1
					break
				fi
			done
			if [ "$ok" -eq 0 ]; then
				real_args+=("$arg")
			fi
		fi
	done

	# if [[ "$is_dap" == "1" ]]; then
	# 	debug=1
	# fi

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

	raw_args="${real_go_binary} test -v -vet=all -json -cover $extra_args ${real_args[*]} $target_dir ${adjusted_runtime_args[*]}"

	if [[ "$is_dap" == "1" ]]; then
		export CGO_LDFLAGS="-Wl,-no_warn_duplicate_libraries" # dap will fail otherwise

		raw_args="${real_go_binary} test -o '$output_file' -gcflags='${gcflags_arg}' ${real_args[*]} "
		# raw_args+=" && ls -laR $output_file && [ -f \"$output_file\" ] "
		if [[ "$codesign" == "1" ]]; then
			raw_args+=" && ${real_go_binary} run $my_absolute_dir/cmd/codesign -skip-run -file=$output_file "
		fi
		# raw_args+=" && dlv exec --listen=:2347 --api-version=2 --continue=false $output_file.test"

	elif [[ "$debug" == "1" ]]; then

		items=$(safe_go list -f '{{.ImportPath}}' "$target_dir")
		# if there are more than one item, throw an error
		if [[ $(echo "$items" | wc -l) -gt 1 ]]; then
			echo "Error: more than one item in target directory"
			exit 1
		fi
		# item="$items"

		# go test -c -o /tmp/test.test "$item"

		# project_root_dir="$(dirname -- "${BASH_SOURCE[0]}")"
		raw_args=""
		tmpdir=$(mktemp -d)
		remove_tmpdir() {
			rm -rf "$tmpdir"
		}
		trap remove_tmpdir EXIT
		for item in $items; do
			module=$item
			file_name=$(basename "$module")
			# raw_args+="go test -c -o $tmpdir -v -vet=all -json -cover $extra_args ${real_args[*]} $module"
			raw_args+="${real_go_binary} test -c -o $tmpdir -gcflags=\"all=-N -l\" $extra_args ${real_args[*]} $module"
			raw_args+=" && [ -f \"$tmpdir/$file_name.test\" ] "
			if [[ "$codesign" == "1" ]]; then
				# raw_args+=" && codesign --entitlements $entitlements_file --verbose=0 -s - $tmpdir/$file_name.test "
				raw_args+=" && ${real_go_binary} run $my_absolute_dir/cmd/codesign -skip-run -file=$tmpdir/$file_name.test "
			fi
			# 		launchConfig := map[string]interface{}{
			#     "type":    "go",
			#     "request": "launch",
			#     "name":    "Debug VF Tests",
			#     // “exec” mode makes the Go extension invoke dlv exec under the hood
			#     "mode":    "exec",
			#     "program": fmt.Sprintf("%s/%s", getCwd(), bin),
			#     "args": []string{
			#         "--headless",
			#         "--listen=:2347",
			#         "--api-version=2",
			#         "--accept-multiclient",
			#         "--continue=false",
			#     },
			# }
			# raw_args+=" && go tool test2json -t -p $module -- $tmpdir/$file_name.test -test.v ${adjusted_runtime_args[*]} || true; "
			# start the delv server
			# dlv dap --headless --listen=:2347 --api-version=2 &
			# run debugger (continue? or -- )
			# add this file to path for the dlv command
			# export PATH="$PATH:$my_absolute_dir"
			# uri='cursor://fabiospampinato.vscode-debug-launcher/launch?args={"type":"go","request":"attach","name":"Debug VF Tests","mode":"remote","host":"127.0.0.1","port":2347,"program":"'"$tmpdir/$file_name.test"'"}'
			raw_args+=" && dlv exec  --listen=:2347 --api-version=2 --continue=false $tmpdir/$file_name.test"
			# raw_args+=" && open $uri"
		done
	elif [[ "$codesign" == "1" ]]; then
		extra_args+="-exec 'go run $my_absolute_dir/cmd/codesign'"
	fi
	if [[ "$vv" == "1" || "$debug" == "1" ]]; then
		# if ! [[ "$is_dap" == "1" ]]; then
		echo -e "calling: $raw_args"
		# fi
	fi

	if [[ "$ide" == "1" || "$debug" == "1" || "$is_dap" == "1" ]]; then
		echo "running: $raw_args"
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
	# echo "tool: $*"
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

	safe_go tool "$@"

	exit $?
fi

# otherwise run go directly with all arguments
safe_go "$@"
