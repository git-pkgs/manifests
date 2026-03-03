;;; Guix manifest for development environment.

(specifications->manifest
 (list
  "bash"
  "coreutils"
  "gcc-toolchain"
  "git-minimal"
  "grep"
  "gzip"
  "make"
  ;; "commented-out-package"
  "nss-certs"
  "pkg-config"
  "python"
  "sed"
  "tar"
  "util-linux"
  "wget"
  "xz"))
