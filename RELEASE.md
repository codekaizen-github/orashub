# Release Process

This document outlines the steps to create a new release of the WordPress Plugin Registry ORAS tool.

## Release Checklist

1. **Update Version Information**
   - Update any version constants in the code
   - Ensure CHANGELOG.md is up to date with all notable changes

2. **Run Tests**
   ```bash
   make test
   ```

3. **Create a Local Build and Test**
   ```bash
   make build
   # Test the binary works as expected
   ./wordpress-plugin-registry-oras --version
   ```

4. **Create and Push a Git Tag**
   ```bash
   # For a new version, e.g., v1.0.0
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

5. **Monitor GitHub Actions**
   - Watch the GitHub Actions workflow to ensure it completes successfully
   - The workflow will automatically:
     - Build binaries for all supported platforms
     - Create a GitHub Release
     - Upload the binaries to the release

6. **Verify the Release**
   - Check that all binaries are available on the GitHub Releases page
   - Download and test at least one binary to ensure it works correctly

## Release Naming Convention

- Use semantic versioning (MAJOR.MINOR.PATCH)
- Prefix version tags with "v" (e.g., v1.0.0)

## Hotfix Process

If you need to create a hotfix:

1. Create a branch from the tagged release:
   ```bash
   git checkout -b hotfix/v1.0.1 v1.0.0
   ```

2. Make your fixes and commit them

3. Tag the hotfix and push:
   ```bash
   git tag -a v1.0.1 -m "Hotfix v1.0.1"
   git push origin v1.0.1
   ```
