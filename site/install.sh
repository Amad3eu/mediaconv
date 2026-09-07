#!/bin/sh
set -eu

repo="Amad3eu/mediaconv"
project="mediaconv"
install_dir="${MEDIACONV_INSTALL_DIR:-}"
version="${MEDIACONV_VERSION:-latest}"
github_base="https://github.com/${repo}"

say() {
  printf '%s\n' "$*"
}

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

has() {
  command -v "$1" >/dev/null 2>&1
}

download() {
  url="$1"
  output="$2"

  if has curl; then
    curl -fsSL "$url" -o "$output"
    return
  fi

  if has wget; then
    wget -q "$url" -O "$output"
    return
  fi

  die "curl or wget is required"
}

latest_tag() {
  latest_url="${github_base}/releases/latest"

  if has curl; then
    effective_url="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$latest_url")"
  elif has wget; then
    effective_url="$(wget -qO- --max-redirect=0 "$latest_url" 2>&1 \
      | sed -n 's/.*Location: //p' \
      | tr -d '\r' \
      | tail -n 1)"
  else
    die "curl or wget is required"
  fi

  tag="$(printf '%s' "$effective_url" | sed 's#.*/tag/##')"
  case "$tag" in
    v[0-9]*.[0-9]*.[0-9]*)
      printf '%s\n' "$tag"
      ;;
    *)
      die "could not resolve latest release tag"
      ;;
  esac
}

os_name() {
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    linux)
      printf 'linux\n'
      ;;
    darwin)
      printf 'darwin\n'
      ;;
    *)
      die "unsupported operating system: $os"
      ;;
  esac
}

arch_name() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64 | amd64)
      printf 'amd64\n'
      ;;
    arm64 | aarch64)
      printf 'arm64\n'
      ;;
    *)
      die "unsupported architecture: $arch"
      ;;
  esac
}

verify_checksum() {
  file="$1"
  checksums="$2"
  name="$(basename "$file")"

  expected="$(grep "  ${name}\$" "$checksums" || true)"
  if [ -z "$expected" ]; then
    die "checksum for ${name} was not found"
  fi

  if has sha256sum; then
    printf '%s\n' "$expected" | (cd "$(dirname "$file")" && sha256sum -c -)
    return
  fi

  if has shasum; then
    expected_hash="$(printf '%s\n' "$expected" | awk '{print $1}')"
    actual_hash="$(shasum -a 256 "$file" | awk '{print $1}')"
    [ "$expected_hash" = "$actual_hash" ] || die "checksum verification failed"
    say "${name}: OK"
    return
  fi

  die "sha256sum or shasum is required to verify downloads"
}

choose_install_dir() {
  if [ -n "$install_dir" ]; then
    printf '%s\n' "$install_dir"
    return
  fi

  if [ -w /usr/local/bin ] || has sudo; then
    printf '/usr/local/bin\n'
    return
  fi

  if [ -n "${HOME:-}" ]; then
    printf '%s/.local/bin\n' "$HOME"
    return
  fi

  die "set MEDIACONV_INSTALL_DIR to choose an install directory"
}

copy_binary() {
  src="$1"
  dst="$2"
  dir="$(dirname "$dst")"

  if [ -d "$dir" ] && [ -w "$dir" ]; then
    if has install; then
      install -m 0755 "$src" "$dst"
    else
      cp "$src" "$dst"
      chmod 0755 "$dst"
    fi
    return
  fi

  if [ ! -d "$dir" ]; then
    if has sudo; then
      sudo mkdir -p "$dir"
    else
      mkdir -p "$dir"
    fi
  fi

  if [ -w "$dir" ]; then
    if has install; then
      install -m 0755 "$src" "$dst"
    else
      cp "$src" "$dst"
      chmod 0755 "$dst"
    fi
  elif has sudo; then
    if has install; then
      sudo install -m 0755 "$src" "$dst"
    else
      sudo cp "$src" "$dst"
      sudo chmod 0755 "$dst"
    fi
  else
    die "cannot write to $dir; set MEDIACONV_INSTALL_DIR to a writable path"
  fi
}

make_temp_dir() {
  base="${TMPDIR:-/tmp}"

  if has mktemp; then
    mktemp -d "${base%/}/${project}-install.XXXXXX"
    return
  fi

  fallback="${base%/}/${project}-install.$$"
  mkdir -p "$fallback"
  printf '%s\n' "$fallback"
}

if [ "$version" = "latest" ]; then
  tag="$(latest_tag)"
else
  tag="$version"
fi

case "$tag" in
  v[0-9]*.[0-9]*.[0-9]*)
    ;;
  *)
    die "MEDIACONV_VERSION must look like v1.2.3"
    ;;
esac

plain_version="${tag#v}"
os="$(os_name)"
arch="$(arch_name)"
archive="${project}_${plain_version}_${os}_${arch}.tar.gz"
checksums="${project}_${plain_version}_checksums.txt"
download_base="${github_base}/releases/download/${tag}"
tmp="$(make_temp_dir)"
bin_dir="$(choose_install_dir)"
target="${bin_dir}/${project}"

trap 'rm -rf "$tmp"' EXIT INT TERM

say "Installing ${project} ${tag} for ${os}/${arch}"
download "${download_base}/${archive}" "${tmp}/${archive}"
download "${download_base}/${checksums}" "${tmp}/${checksums}"
verify_checksum "${tmp}/${archive}" "${tmp}/${checksums}"

tar -xzf "${tmp}/${archive}" -C "$tmp"
[ -x "${tmp}/${project}" ] || die "archive did not contain ${project}"

copy_binary "${tmp}/${project}" "$target"

case ":$PATH:" in
  *:"$bin_dir":*)
    ;;
  *)
    say "warning: ${bin_dir} is not in PATH"
    say "add it to PATH before running ${project}"
    ;;
esac

say "Installed: $target"
"$target" version

if ! has ffmpeg || ! has ffprobe; then
  say "warning: ffmpeg and ffprobe are required for conversions"
  say "run '${project} doctor' after installing FFmpeg"
fi
