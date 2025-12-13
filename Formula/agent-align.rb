class AgentAlign < Formula
  desc "Keep MCP configuration files aligned across coding agents"
  homepage "https://github.com/timbuchinger/agent-align"
  version "VERSION_PLACEHOLDER"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/timbuchinger/agent-align/releases/download/vVERSION_PLACEHOLDER/agent-align-darwin-arm64.tar.gz"
      sha256 "SHA256_ARM64_PLACEHOLDER"
    else
      url "https://github.com/timbuchinger/agent-align/releases/download/vVERSION_PLACEHOLDER/agent-align-darwin-amd64.tar.gz"
      sha256 "SHA256_AMD64_PLACEHOLDER"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/timbuchinger/agent-align/releases/download/vVERSION_PLACEHOLDER/agent-align-linux-arm64.tar.gz"
      sha256 "SHA256_LINUX_ARM64_PLACEHOLDER"
    else
      url "https://github.com/timbuchinger/agent-align/releases/download/vVERSION_PLACEHOLDER/agent-align-linux-amd64.tar.gz"
      sha256 "SHA256_LINUX_AMD64_PLACEHOLDER"
    end
  end

  def install
    if OS.mac?
      if Hardware::CPU.arm?
        bin.install "agent-align-darwin-arm64" => "agent-align"
      else
        bin.install "agent-align-darwin-amd64" => "agent-align"
      end
    else
      if Hardware::CPU.arm?
        bin.install "agent-align-linux-arm64" => "agent-align"
      else
        bin.install "agent-align-linux-amd64" => "agent-align"
      end
    end
  end

  test do
    system "#{bin}/agent-align", "--version"
  end
end
