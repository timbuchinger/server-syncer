# Homebrew Distribution Setup

This document describes how to set up and maintain the Homebrew tap for
agent-align.

## Overview

agent-align can be installed via Homebrew using a custom tap. The release
pipeline automatically updates the Homebrew formula when a new version is
released.

## Installation for Users

Once the tap is set up, users can install agent-align using:

```bash
brew tap timbuchinger/agent-align
brew install agent-align
```

Or in a single command:

```bash
brew install timbuchinger/agent-align/agent-align
```

## Setting Up the Homebrew Tap Repository

### 1. Create the Homebrew Tap Repository

Create a new repository named `homebrew-agent-align` in your GitHub account.
The repository must follow the naming convention `homebrew-*` for Homebrew to
recognize it as a tap.

```bash
# Initialize the tap repository
mkdir homebrew-agent-align
cd homebrew-agent-align
git init
mkdir Formula
# Add a README
echo "# Homebrew Tap for agent-align" > README.md
git add README.md
git commit -m "Initial commit"
git branch -M main
git remote add origin https://github.com/timbuchinger/homebrew-agent-align.git
git push -u origin main
```

### 2. Configure GitHub Secrets and Variables

In the **agent-align** repository (this repository), configure the following:

#### Required Secrets

1. **HOMEBREW_TAP_TOKEN**: A GitHub Personal Access Token (PAT) with
   permission to push to the homebrew tap repository.

   To create this token:
   - Go to GitHub Settings → Developer settings → Personal access tokens →
     Tokens (classic)
   - Click "Generate new token (classic)"
   - Give it a descriptive name like "agent-align homebrew tap updater"
   - Select scopes:
     - `repo` (Full control of private repositories) - needed to push to the
       tap repository
   - Click "Generate token"
   - Copy the token and add it as a secret named `HOMEBREW_TAP_TOKEN` in the
     agent-align repository settings

   **Alternative**: Use a fine-grained personal access token:
   - Go to GitHub Settings → Developer settings → Personal access tokens →
     Fine-grained tokens
   - Click "Generate new token"
   - Give it a descriptive name
   - Select "Only select repositories" and choose `homebrew-agent-align`
   - Under "Repository permissions", set:
     - Contents: Read and write
   - Generate and copy the token

#### Required Variables

1. **HOMEBREW_TAP_REPO**: The full repository name for the homebrew tap in
   the format `owner/repo`.

   Example: `timbuchinger/homebrew-agent-align`

   To add this variable:
   - Go to the agent-align repository Settings → Secrets and variables →
     Actions → Variables tab
   - Click "New repository variable"
   - Name: `HOMEBREW_TAP_REPO`
   - Value: `timbuchinger/homebrew-agent-align` (adjust owner as needed)

### 3. Test the Setup

After configuring the secrets and variables, trigger a release to test the
Homebrew formula update:

1. Ensure you have commits that warrant a release
2. Go to Actions → Release workflow
3. Click "Run workflow"
4. Select the release type
5. Monitor the workflow to ensure it completes successfully
6. Check the homebrew tap repository to verify the formula was updated

### 4. Verify the Formula

After a release, verify the formula works:

```bash
brew tap timbuchinger/agent-align
brew install agent-align
agent-align --version
```

## Troubleshooting

### Formula Update Fails

If the "Push Homebrew formula to tap" step fails:

1. Verify `HOMEBREW_TAP_TOKEN` has the correct permissions
2. Verify `HOMEBREW_TAP_REPO` is set to the correct repository name
3. Ensure the tap repository exists and has a `Formula/` directory
4. Check the workflow logs for specific error messages

### Formula Not Found After Installation

If `brew install` says the formula isn't found:

1. Ensure the tap repository name follows the `homebrew-*` pattern
2. Verify the formula file is in the `Formula/` directory
3. Check that the formula file is named `agent-align.rb`

### Version Mismatch

If the installed version doesn't match the latest release:

1. Run `brew update` to refresh Homebrew
2. Run `brew upgrade agent-align`
3. Check the formula in the tap repository to ensure it was updated

## Skipping Homebrew Updates

If `HOMEBREW_TAP_TOKEN` or `HOMEBREW_TAP_REPO` are not set, the workflow will
skip the Homebrew tap update step without failing. This allows the release
process to work even if Homebrew distribution is not configured yet.
