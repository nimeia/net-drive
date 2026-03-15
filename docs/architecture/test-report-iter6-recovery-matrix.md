# Iter 6 Recovery Test Matrix Report

Generated: 2026-03-15T04:01:20.638680+00:00

## build
returncode: 0

stdout:
```

```

stderr:
```

```

## session_unit
returncode: 0

stdout:
```
ok  	developer-mount/internal/server	0.015s

```

stderr:
```

```

## recovery_integration
returncode: 0

stdout:
```
=== RUN   TestRecoveryResumeSessionMatrix
=== RUN   TestRecoveryResumeSessionMatrix/success
2026/03/15 04:01:17 devmount server listening on 127.0.0.1:32861 root=/tmp/TestRecoveryResumeSessionMatrixsuccess1816523520/001
=== RUN   TestRecoveryResumeSessionMatrix/client-instance-mismatch
2026/03/15 04:01:17 devmount server listening on 127.0.0.1:43075 root=/tmp/TestRecoveryResumeSessionMatrixclient-instance-mismatch2336964375/001
=== RUN   TestRecoveryResumeSessionMatrix/expired-session
2026/03/15 04:01:17 devmount server listening on 127.0.0.1:40127 root=/tmp/TestRecoveryResumeSessionMatrixexpired-session3181945550/001
--- PASS: TestRecoveryResumeSessionMatrix (0.01s)
    --- PASS: TestRecoveryResumeSessionMatrix/success (0.00s)
    --- PASS: TestRecoveryResumeSessionMatrix/client-instance-mismatch (0.00s)
    --- PASS: TestRecoveryResumeSessionMatrix/expired-session (0.00s)
=== RUN   TestRecoveryHandleMatrix
2026/03/15 04:01:17 devmount server listening on 127.0.0.1:38105 root=/tmp/TestRecoveryHandleMatrix4199857449/001
--- PASS: TestRecoveryHandleMatrix (0.00s)
=== RUN   TestRecoveryRevalidateAndResubscribeMatrix
2026/03/15 04:01:17 devmount server listening on 127.0.0.1:37331 root=/tmp/TestRecoveryRevalidateAndResubscribeMatrix2756901259/001
--- PASS: TestRecoveryRevalidateAndResubscribeMatrix (0.00s)
PASS
ok  	developer-mount/tests/integration	0.035s

```

stderr:
```

```

## watch_integration
returncode: 0

stdout:
```
ok  	developer-mount/tests/integration	0.031s

```

stderr:
```

```

## control_integration
returncode: 0

stdout:
```
ok  	developer-mount/tests/integration	0.026s

```

stderr:
```

```
