{ pkgs }:
{
  qwen25-05b = rec {
    name = "qwen2.5-0.5b-instruct-q4_k_m.gguf";
    sha256 = "74a4da8c9fdbcd15bd1f6d01d621410d31c6fc00986f5eb687824e7b93d7a9db";
    size = 491400032;

    # Hub/test-driver side only: never reference `file` inside a site NixOS closure.
    file = pkgs.fetchurl {
      url = "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/9217f5db79a29953eb74d5343926648285ec7e67/${name}";
      inherit sha256;
    };
  };
}
