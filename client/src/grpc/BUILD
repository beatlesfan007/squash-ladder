load("@aspect_rules_ts//ts:defs.bzl", "ts_project")
load("@rules_proto_grpc_js//:defs.bzl", "js_proto_library", "js_grpc_web_library")

# Generate JavaScript/TypeScript code from proto files using rules_proto_grpc
# js_proto_library generates both JavaScript and TypeScript definitions
js_proto_library(
    name = "ladder_proto_ts",
    protos = ["//server/proto:ladder_proto"],
)

# Generate gRPC-Web service client code from proto files
# This generates the service stubs for making gRPC-Web calls
js_grpc_web_library(
    name = "ladder_grpc_web",
    protos = ["//server/proto:ladder_proto"],
    deps = [
        ":ladder_proto_ts",
    ],
)

# TypeScript project that uses the generated proto files
ts_project(
    name = "grpc_ts",
    tsconfig = "//client:tsconfig.json",
    srcs = [
        "ladderService.ts",
    ],
    deps = [
        ":ladder_proto_ts",
        ":ladder_grpc_web",
        "@npm//@improbable-eng/grpc-web",
        "@npm//google-protobuf",
    ],
    visibility = ["//visibility:public"],
)
