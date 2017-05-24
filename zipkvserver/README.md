# zipkvserver

[S3 compatible][s3c] backend for [S3QL][s3ql] storing files in
large blocks.

```
$ go get github.com/starius/invisiblefs/zipkvserver
```

S3QL is a file system based on FUSE which stores data
using storage services. It supports multiple backends, in particular
[S3 compatible][s3c]. This tool is an implementation of such a backend.

The tool concatenates stored objects together up to maximum size
(default is 40 MiB) and stores the results in files `block<id>`.
The map from a object name to a location in block files is stored
in gzipped protobuf in file `db<id>`. Blocks are not removed and
not changed after creation, but the old index is removed after
writting it to new file (which happens when a new block is created
and in the end). S3QL uses metadata feature of S3 so zipkvserver also
supports it - all the metadata is stored in the index. The index is
stored in form of history of `PUT` and `DELETE` operations,
so the storage can be rolled back to any operation. See tools in
directories `zipkvhistory` and `zipkvrebase` for this.

S3QL encrypts the data before sending it to a backend. But it caches
plaintext data in a directory specified in `--cachedir` option.
By default this option is set to `~/.s3ql`. To avoid plaintext data
reaching drive, put cachedir to `tmpfs` and disable swap.

```
$ mkdir /tmp/cache
$ sudo swapoff -a
```

Run the tool:

```
$ mkdir zipdir
$ zipkvserver -dir zipdir
```

The data will be stored in directory `zipdir`.

Maximum block size (in bytes) can be changed with `-bs` option.
To get list of other options run it with `-h`.

By default it runs on `127.0.0.1:7711/bucket` which we'll
use in the commands below.

Create S3QL filesystem:

```
$ mkfs.s3ql --cachedir=/tmp/cache --max-obj-size=4000 --backend-options=no-ssl s3c://127.0.0.1:7711/bucket
```

It'll ask you for backend login and passphrase - put anything there.
Then it'll ask you for the encryption password twice.

Mount S3QL filesystem:

```
$ mkdir mountpoint
$ mount.s3ql --cachedir=/tmp/cache --cachesize=20000 --metadata-upload-interval=300 --backend-options=no-ssl s3c://127.0.0.1:7711/bucket mountpoint
```

We specified `--metadata-upload-interval=300` to reduce data loss
in case of system crashing.

Now you can work with the files in the directory `mountpoint`.
Read about [advanced S3QL features][s3ql-advanced] to know about
cheap snapshots, fast recursive removal and immutable directories.

After finishing your work with the directory you have to unmount it
properly and then stop `zipkvserver` to prevent data loss:

```
$ umount.s3ql mountpoint
# Type Ctrl+C in the terminal with zipkvserver running.
```

[s3c]: http://www.rath.org/s3ql-docs/backends.html#s3-compatible
[s3ql]: http://www.rath.org/s3ql-docs/
[s3ql-advanced]: http://www.rath.org/s3ql-docs/special.html
