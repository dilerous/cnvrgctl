# cnvrgctl
cnvrg.io delivery cli tool

## How to Install

#### Install the binary
1. Download the binary from `releases`.

`https://github.com/dilerous/cnvrgctl/releases`

2. Untar the file and move to your `$PATH`.

`tar -xzf <distro-architecture>.tar.gz`

`mv cnvrgctl /usr/local/bin`

3. `cnvrgctl --help`

#### Install using Homebrew
1. `brew tap dilerous/homebrew-dilerous`

2. `brew install cnvrgctl`

3. `cnvrgctl --help`

#### Build from Source
1. Download and install golang.

   `https://go.dev/doc/install`
2. Clone down the directory.

   `git clone https://github.com/dilerous/cnvrgctl.git && cd cnvrgctl`
3. Run the make file to create the go binary

   `make mac` #Create binary for arm on MacOS

   `make linux` #Create binary for amd64 on Linux

4. `cnvrgctl --help`

## How to Use
1. Run cnvrgctl as a normal cli tool

2. `cnvrgctl --help` to bring up the help menu to navigate available commands.

#### Backup sub-command
Run `cnvrgctl backup` to backup the current cnvrg.io installation. This includes both the files and the Postgres database.

Example:

`cnvrgctl backup files -n cnvrg` This will backup the minio `cnvrg-storage` bucket locally to be migrated to new installs.

#### Restore sub-command
Run `cnvrgctl restore` to restore either files or the Postgres database to your new installation of cnvrg.io

Example:

Run `cnvrgctl restore files -n cnvrg` to restore the local files in the `./cnvrg-storage` folder to your new cnvrg.io installation.

#### Logs sub-command
Run `cnvrgctl logs` to pull all logs from the running pods in the namespace selected.

Example:

Run `cnvrgctl logs -n cnvrg` to grab all logs from the namespace and output the logs to a local folder called `./logs`.

#### Install sub-command
Run `cnvrgctl install` to deploy ArgoCD, minio operator and a tenant, nginx, or sealed secrets.

Example:

Run `cnvrgctl -n argocd install argocd -d argocd.dilerous.cloud` to install ArgoCD in the `argocd` namespace while setting the ingress host to `argocd.dilerous.cloud`.

## Minio
How to connect to the minio bucket using the `mc` cli tool.
1. Following the deployment of the operator and tenant you need to set an alias
   to access the minio bucket. The default bucket created during installation is
   `cnvrg-backups`.
2. Download the minio cli tool called `mc`.
    ```
    curl https://dl.min.io/client/mc/release/linux-amd64/mc \
    --create-dirs \
    -o $HOME/minio-binaries/mc

    chmod +x $HOME/minio-binaries/mc
    export PATH=$PATH:$HOME/minio-binaries/
    ```
3. Set the alias, update the minio url as needed to match your deployment. The
   username and password are the default. In the minio console up can creat an
   access and secret key for additional security.

    `mc alias set backup https://minio.aws.dilerous.cloud minio minio123 --insecure`
4. To list files in your bucket run:

    `mc ls backup/cnvrg-backups --insecure`
5. Congrats! You can successfully manage your bucket with the `mc` cli tool.
6. If the minio environment is not going to be deleted following the migration,
   you need to change the default minio password to reduce security vunerabilties.
