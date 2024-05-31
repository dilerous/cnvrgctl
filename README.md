# cnvrgctl
cnvrg.io delivery cli tool


### Minio
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
