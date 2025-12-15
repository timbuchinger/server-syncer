#!/usr/bin/env ruby
require 'erb'
require 'fileutils'

root = File.expand_path('..', __dir__)
# Prefer the tapped repo under the workspace if present (this script may be
# invoked from within the freshly-cloned tap directory in CI). Determine a
# sensible tap directory and template location.
possible_tap_paths = [
  File.join(root, '..', 'homebrew-tap'),        # sibling in workspace
  File.join(Dir.pwd),                           # current working directory (when run from tap)
]

tap_dir = possible_tap_paths.find { |p| File.exist?(p) && File.directory?(p) }

template_path = tap_dir ? File.join(tap_dir, 'Formula', 'agent-align.rb.erb') : nil
output_path = if tap_dir
  File.join(tap_dir, 'Formula', 'agent-align.rb')
else
  File.join(root, '..', 'homebrew-tap', 'Formula', 'agent-align.rb')
end

# Fallback embedded template if no template file exists in the tap
EMBEDDED_TEMPLATE = <<~'ERB'
class AgentAlign < Formula
  desc "Sync MCP configs across coding agents"
  homepage "https://github.com/timbuchinger/agent-align"
  version "<%= ver %>"

  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/timbuchinger/agent-align/releases/download/v#{version}/agent-align-darwin-arm64.tar.gz"
      sha256 "<%= darwin_arm_sha %>"
    end
    on_intel do
      url "https://github.com/timbuchinger/agent-align/releases/download/v#{version}/agent-align-darwin-amd64.tar.gz"
      sha256 "<%= darwin_amd_sha %>"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/timbuchinger/agent-align/releases/download/v#{version}/agent-align-linux-amd64.tar.gz"
      sha256 "<%= linux_amd_sha %>"
    end
    on_arm do
      url "https://github.com/timbuchinger/agent-align/releases/download/v#{version}/agent-align-linux-arm64.tar.gz"
      sha256 "<%= linux_arm_sha %>"
    end
  end

  def install
    bin.install "agent-align"
    prefix.install_metafiles
  end

  test do
    assert_predicate bin/"agent-align", :exist?
    assert_predicate bin/"agent-align", :executable?
  end
end
ERB

template = if template_path && File.exist?(template_path)
  File.read(template_path)
else
  EMBEDDED_TEMPLATE
end
ver = ENV['VER'] || abort("VER not set")
darwin_arm_sha = ENV['DARWIN_ARM_SHA'] || ''
darwin_amd_sha = ENV['DARWIN_AMD_SHA'] || ''
linux_amd_sha = ENV['LINUX_AMD_SHA'] || ''
linux_arm_sha = ENV['LINUX_ARM_SHA'] || ''

renderer = ERB.new(template, trim_mode: '-')
result = renderer.result(binding)

# Ensure directory exists
FileUtils.mkdir_p(File.dirname(output_path))
File.write(output_path, result)
puts "Wrote formula to #{output_path} (VER=#{ver})"
