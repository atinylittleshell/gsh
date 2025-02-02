# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class Gsh < Formula
  desc "A modern, POSIX-compatible, generative shell"
  homepage "https://github.com/atinylittleshell/gsh"
  version "0.20.1"
  license "GPL-3.0-or-later"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/atinylittleshell/gsh/releases/download/v0.20.1/gsh_Darwin_x86_64.tar.gz"
      sha256 "f68c9a03ebe58d854844e3f2a04886f749b5c49a0f340b539672dd66b7d9cabc"

      def install
        bin.install "gsh"
      end
    end
    if Hardware::CPU.arm?
      url "https://github.com/atinylittleshell/gsh/releases/download/v0.20.1/gsh_Darwin_arm64.tar.gz"
      sha256 "6ab97e14623e5ec27f05df6da8e40b5f85f7eaafb98c5aaad89b2014584c50a0"

      def install
        bin.install "gsh"
      end
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      if Hardware::CPU.is_64_bit?
        url "https://github.com/atinylittleshell/gsh/releases/download/v0.20.1/gsh_Linux_x86_64.tar.gz"
        sha256 "d33d07b2027d87c1f0f097d84bbdc17d435f71a3e79392730239b0d0c2f5278f"

        def install
          bin.install "gsh"
        end
      end
    end
    if Hardware::CPU.arm?
      if Hardware::CPU.is_64_bit?
        url "https://github.com/atinylittleshell/gsh/releases/download/v0.20.1/gsh_Linux_arm64.tar.gz"
        sha256 "f00349a1177cce4d54ca54ced5d1d5619c9cde2b92f1b5bf432b6845004bb015"

        def install
          bin.install "gsh"
        end
      end
    end
  end
end
