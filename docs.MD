# YABD (Yet Another Bunch of Docs)

### Get Nexus to autobackup its DB to S3:

*TL;DR: Set environment variables.*
```shell
export AWS_ACCESS_KEY_ID="..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_REGION="ap-northwest-2"
export AWS_BACKUP_BUCKET_NAME="..."
export AWS_BACKUP_PATH=".." # Path in the bucket to push the backup to
```

### Federated identity: Read user information from other systems:

1. Create a CA to pin the client's identity: `./maintenance-tool CREATECA`
2. Mint a client cert & key: `./maintenance-tool MINTCLIENTCERT`
3. Copy the client cert and key to the client.
4. Start launching the server with additional flags: `--federation-cert <path-to-CA-cert> --federation-enabled`
5. You can now make HTTPS requests to get the list of users. Use the client cert. Make requests to `/federation/v1/accounts/users`.

You can test the endpoint using: `./maintenance-tool GETUSERLISTREMOTE`.

### App-specific

#### Enable ytdl

1. Install `youtube-dl` and `ffmpeg`.
2. For the users who should have access, add an attribute to their account named `ytdl` with value `allow`.
3. Make sure they have a fs source  `/music`.
