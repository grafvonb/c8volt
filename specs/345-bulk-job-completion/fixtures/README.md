# Static cancellation-listener reference models

These BPMN files were prepared before the user excluded real-environment testing from implementation. Retain them as static examples only. Do not deploy, start, cancel or clean up resources from them during implementation or Ralph iterations.

The parent describes three parallel child instances; the child describes a native user task with a canceling listener. `__RUN_ID__` and `listenerType` are symbolic model inputs. Local XML checks can validate references and structure, not engine semantics.

SC-006 is verified by `cmd/update_job_batch_listener_test.go` against test-owned HTTP fakes. These files are not required to run that test. No resources were created from them, so no live cleanup is required.
