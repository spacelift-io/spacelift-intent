// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

//go:generate mkdir -p ../generated/tfplugin5
//go:generate protoc --go_out=paths=source_relative:../generated/tfplugin5 --go-grpc_out=paths=source_relative:../generated/tfplugin5 --proto_path=../proto ../proto/tfplugin5.proto

package tools
