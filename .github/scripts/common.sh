#!/usr/bin/env sh
set -eu

# Compute hash for a file using available tools
# Usage: compute_hash <file> <algorithm>
# algorithm: sha256 or sha512
compute_hash() {
    file_path="${1}"
    algorithm="${2}"

    if [ ! -f "${file_path}" ]; then
        echo "Error: File not found: ${file_path}" >&2
        return 1
    fi

    # Try sha256sum/sha512sum first (GNU coreutils)
    if command -v "${algorithm}sum" >/dev/null 2>&1; then
        "${algorithm}sum" "${file_path}" | cut -d' ' -f1
        return 0
    fi

    # Try shasum (macOS/BSD)
    if command -v shasum >/dev/null 2>&1; then
        case "${algorithm}" in
            sha256)
                shasum -a 256 "${file_path}" | cut -d' ' -f1
                return 0
                ;;
            sha512)
                shasum -a 512 "${file_path}" | cut -d' ' -f1
                return 0
                ;;
        esac
    fi

    # Try openssl as fallback
    if command -v openssl >/dev/null 2>&1; then
        openssl dgst "-${algorithm}" "${file_path}" | cut -d' ' -f2
        return 0
    fi

    echo "Error: No hash tool available for ${algorithm}" >&2
    return 1
}

# Detect hash algorithm from hash length
# 32 chars = MD5, 64 chars = SHA256, 128 chars = SHA512
detect_algorithm() {
    hash="${1}"
    hash_len="${#hash}"

    case "${hash_len}" in
        32)
            echo "md5"
            ;;
        64)
            echo "sha256"
            ;;
        128)
            echo "sha512"
            ;;
        *)
            echo "Error: Unknown hash length: ${hash_len}" >&2
            return 1
            ;;
    esac
}

# Usage: fetch_checksum_content <checksum_ref>
fetch_checksum_content() {
    checksum_ref="${1}"

    # HTTPS URL
    case "${checksum_ref}" in
        https://*)
            if command -v curl >/dev/null 2>&1; then
                curl --fail --location --silent --show-error "${checksum_ref}"
                return $?
            elif command -v wget >/dev/null 2>&1; then
                wget -q -O - "${checksum_ref}"
                return $?
            else
                echo "Error: No tool available to fetch HTTPS URL" >&2
                return 1
            fi
            ;;
        http://*)
            echo "Error: HTTP URLs not allowed for checksums (HTTPS required)" >&2
            return 1
            ;;
        *)
            echo "Error: Checksum ref must be an HTTPS URL: ${checksum_ref}" >&2
            return 1
            ;;
    esac
}

# Extract expected hash from checksum content
# Supports: plain hash, GNU format, BSD format
# Usage: extract_hash <checksum_content> <target_basename>
extract_hash() {
    checksum_content="${1}"
    target_basename="${2}"

    # Strip .tmp.* suffix for temporary file matching
    # Example: haproxy-1.2.3.tar.gz.tmp.12345 -> haproxy-1.2.3.tar.gz
    target_basename="${target_basename%.tmp.*}"

    # If first argument is '-', read from stdin
    if [ "${checksum_content}" = "-" ]; then
        checksum_content=$(cat)
    fi

    # Process line by line and capture first matching hash
    result=$(echo "${checksum_content}" | while IFS= read -r line || [ -n "${line}" ]; do
        # Skip empty lines and comments
        case "${line}" in
            ''|\#*)
                continue
                ;;
        esac

        # BSD format: SHA256 (filename) = HASH
        # Pattern: ALGO (filename) = HASH
        case "${line}" in
            *'('*') = '*)
                rest="${line#* (}"
                file_part="${rest%%) = *}"
                hash_part="${line##* ) = }"

                # Match by basename if multiple entries
                if [ "${file_part}" = "${target_basename}" ] || \
                   [ "${file_part}" = "*${target_basename}" ]; then
                    echo "${hash_part}"
                    exit 0
                fi
                # If single entry, return it
                if [ -z "${target_basename}" ]; then
                    echo "${hash_part}"
                    exit 0
                fi
                continue
                ;;
        esac

        # GNU format: HASH  filename or HASH *filename
        # Also handles plain hash (single token)
        # Split by whitespace (disable glob to prevent * expansion)
        # shellcheck disable=SC2086
        set -f  # Disable glob expansion
        set -- ${line}
        set +f  # Re-enable glob expansion

        if [ $# -ge 1 ]; then
            first_field="${1}"
            # Check if first field looks like a hex hash
            case "${first_field}" in
                *[!0-9a-fA-F]*)
                    # Not a hex string, skip
                    continue
                    ;;
            esac

            # Single token = plain hash
            if [ $# -eq 1 ]; then
                echo "${first_field}"
                exit 0
            fi

            # GNU format: HASH filename
            hash_field="${first_field}"
            file_field="${2}"

            # Remove leading * if present (binary mode indicator)
            file_field="${file_field#\*}"

            # Match by basename
            if [ "${file_field}" = "${target_basename}" ]; then
                echo "${hash_field}"
                exit 0
            fi
        fi
    done)

    if [ -n "${result}" ]; then
        echo "${result}"
        return 0
    fi

    return 1
}

# Verify file checksum against reference
# Usage: verify_checksum <file_path> <checksum_ref>
# checksum_ref: local file path or HTTPS URL
verify_checksum() {
    file_path="${1}"
    checksum_ref="${2}"

    if [ ! -f "${file_path}" ]; then
        echo "Error: File not found: ${file_path}" >&2
        return 1
    fi

    if [ -z "${checksum_ref}" ]; then
        echo "Error: Checksum reference is empty" >&2
        return 1
    fi

    # Fetch checksum content
    checksum_content=$(fetch_checksum_content "${checksum_ref}") || {
        echo "Error: Failed to fetch checksum from: ${checksum_ref}" >&2
        return 1
    }

    if [ -z "${checksum_content}" ]; then
        echo "Error: Checksum content is empty: ${checksum_ref}" >&2
        return 1
    fi

    # Extract target basename for matching
    # Strip .tmp.* suffix if present (from download_tarball temporary files)
    target_basename=$(basename "${file_path}" | sed 's/\.tmp\.[0-9]*$//')
    # Extract expected hash
    expected_hash=$(extract_hash "${checksum_content}" "${target_basename}") || {
        # Try without basename matching (single entry files)
        expected_hash=$(extract_hash "${checksum_content}" "") || {
            echo "Error: Could not extract hash from checksum file for: ${target_basename}" >&2
            return 1
        }
    }

    if [ -z "${expected_hash}" ]; then
        echo "Error: No matching hash found for: ${target_basename}" >&2
        return 1
    fi

    # Detect algorithm from hash length
    algorithm=$(detect_algorithm "${expected_hash}") || {
        echo "Error: Could not detect algorithm for hash: ${expected_hash}" >&2
        return 1
    }

    # Compute actual hash (only sha256/sha512 supported, md5 not supported for verification)
    case "${algorithm}" in
        sha256|sha512)
            actual_hash=$(compute_hash "${file_path}" "${algorithm}") || {
                echo "Error: Failed to compute ${algorithm} hash" >&2
                return 1
            }
            ;;
        md5)
            echo "Error: MD5 checksums not supported for security reasons" >&2
            return 1
            ;;
        *)
            echo "Error: Unsupported algorithm: ${algorithm}" >&2
            return 1
            ;;
    esac

    # Compare hashes (case-insensitive)
    expected_lower=$(echo "${expected_hash}" | tr '[:upper:]' '[:lower:]')
    actual_lower=$(echo "${actual_hash}" | tr '[:upper:]' '[:lower:]')

    if [ "${expected_lower}" != "${actual_lower}" ]; then
        echo "Error: Checksum mismatch for ${file_path}" >&2
        echo "  Expected: ${expected_hash}" >&2
        echo "  Actual:   ${actual_hash}" >&2
        return 1
    fi

    echo "Checksum verified: ${file_path} (${algorithm})" >&2
    return 0
}

download_tarball() {
    url="${1}"
    dest_file="${2}"
    checksum_ref="${3:-}"
    dest_dir="$(dirname "${dest_file}")"
    retries="${DOWNLOAD_RETRIES:-5}"
    connect_timeout="${DOWNLOAD_CONNECT_TIMEOUT:-15}"
    max_time="${DOWNLOAD_MAX_TIME:-300}"
    attempt=1

    mkdir -p "${dest_dir}"
    echo "Downloading ${url} -> ${dest_file}" >&2

    # Check if file exists and verify checksum if provided
    if [ -f "${dest_file}" ]; then
        if [ -n "${checksum_ref}" ]; then
            if verify_checksum "${dest_file}" "${checksum_ref}" 2>/dev/null; then
                echo "File exists and checksum matches: ${dest_file}" >&2
                return 0
            fi
            echo "File exists but checksum mismatch, re-downloading: ${dest_file}" >&2
        else
            echo "File exists (no checksum verification): ${dest_file}" >&2
            return 0
        fi
    fi

    rm -f "${dest_file}"

    while [ "${attempt}" -le "${retries}" ]; do
        tmp_file="${dest_file}.tmp.$$"
        rm -f "${tmp_file}"

        if command -v curl >/dev/null 2>&1; then
            if curl --fail --location --silent --show-error \
                --connect-timeout "${connect_timeout}" \
                --max-time "${max_time}" \
                --output "${tmp_file}" "${url}"; then
                rc=0
            else
                rc=$?
            fi
        else
            if wget -q \
                --connect-timeout="${connect_timeout}" \
                --timeout="${max_time}" \
                -O "${tmp_file}" "${url}"; then
                rc=0
            else
                rc=$?
            fi
        fi

        if [ "${rc}" -eq 0 ]; then
            # Verify checksum if provided
            if [ -n "${checksum_ref}" ]; then
                if verify_checksum "${tmp_file}" "${checksum_ref}"; then
                    mv "${tmp_file}" "${dest_file}"
                    return 0
                fi
                rm -f "${tmp_file}"
                echo "Error: Checksum verification failed for: ${dest_file} (from ${url})" >&2
                return 1
            fi

            # No checksum - just check file is not empty
            if [ -s "${tmp_file}" ]; then
                mv "${tmp_file}" "${dest_file}"
                return 0
            fi

            rm -f "${tmp_file}"
            echo "Error: Downloaded file is empty: ${dest_file} (from ${url})" >&2
            return 1
        fi

        rm -f "${tmp_file}"
        echo "Error: Download failed (attempt ${attempt}/${retries}, exit=${rc}): ${url} -> ${dest_file}" >&2

        attempt=$((attempt + 1))
        if [ "${attempt}" -le "${retries}" ]; then
            sleep 1
        fi
    done

    return 1
}
