# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class Gsh < Formula
  desc "The Generative Shell"
  homepage "https://github.com/atinylittleshell/gsh"
  version "0.5.4"
  license "GPL-3.0"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/atinylittleshell/gsh/releases/download/v0.5.4/gsh_Darwin_x86_64.tar.gz"
      sha256 "3048aaecdf92c3ea32e47fc0ed022d4ea8d3f7636b3595c9ba2969e304ffd50e"

      def install
        bin.install "gsh"
      end
    end
    if Hardware::CPU.arm?
      url "https://github.com/atinylittleshell/gsh/releases/download/v0.5.4/gsh_Darwin_arm64.tar.gz"
      sha256 "d291bfb652f4dd7346cee20f73af986f4f5d6adb1423ff5b8eff6fcbe2412be4"

      def install
        bin.install "gsh"
      end
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      if Hardware::CPU.is_64_bit?
        url "https://github.com/atinylittleshell/gsh/releases/download/v0.5.4/gsh_Linux_x86_64.tar.gz"
        sha256 "fe98be8d153e915bc4f1f47394dd3b57d43bae261386a85b9a05c7551d1f8ae0"

        def install
          bin.install "gsh"
        end
      end
    end
    if Hardware::CPU.arm?
      if Hardware::CPU.is_64_bit?
        url "https://github.com/atinylittleshell/gsh/releases/download/v0.5.4/gsh_Linux_arm64.tar.gz"
        sha256 "ca7177d4bc1abf7b87197b6b00b3bd0b09356b6c3a9337140b8cc7949be8ee79"

        def install
          bin.install "gsh"
        end
      end
    end
  end
end
