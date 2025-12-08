# typed: false
# frozen_string_literal: true

# vStats CLI Formula
# Install: brew install zsai001/vstats/vstats
# Or: brew tap zsai001/vstats && brew install vstats
class Vstats < Formula
  desc "Command-line interface for vStats Cloud server monitoring"
  homepage "https://vstats.zsoft.cc"
  version "1.0.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/zsai001/vstats/releases/download/v#{version}/vstats-cli-darwin-amd64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_FOR_DARWIN_AMD64"
    end

    on_arm do
      url "https://github.com/zsai001/vstats/releases/download/v#{version}/vstats-cli-darwin-arm64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_FOR_DARWIN_ARM64"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/zsai001/vstats/releases/download/v#{version}/vstats-cli-linux-amd64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_FOR_LINUX_AMD64"
    end

    on_arm do
      url "https://github.com/zsai001/vstats/releases/download/v#{version}/vstats-cli-linux-arm64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_FOR_LINUX_ARM64"
    end
  end

  def install
    binary_name = "vstats-cli-#{OS.kernel_name.downcase}-#{Hardware::CPU.arch == :x86_64 ? "amd64" : "arm64"}"
    bin.install binary_name => "vstats"
  end

  def caveats
    <<~EOS
      vStats CLI has been installed!

      Quick Start:
        vstats login              # Login to vStats Cloud
        vstats server list        # List your servers
        vstats server create web1 # Create a new server

      Documentation: https://vstats.zsoft.cc/docs/cli
    EOS
  end

  test do
    assert_match "vstats version", shell_output("#{bin}/vstats version")
  end
end
