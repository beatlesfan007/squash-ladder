load("@aspect_rules_ts//ts:defs.bzl", "ts_project")
load("@rules_nodejs//nodejs:defs.bzl", "nodejs_binary", "npm_package")

# TypeScript project
ts_project(
    name = "client_ts",
    srcs = glob(["src/**/*.ts", "src/**/*.tsx"]),
    tsconfig = "tsconfig.json",
    deps = [
        "@npm//react",
        "@npm//react-dom",
        "//client/src/grpc:grpc_ts",
    ],
)

# NPM package dependencies
npm_package(
    name = "client_package",
    srcs = [
        "package.json",
        ":client_ts",
    ] + glob(["public/**"]),
    package_name = "squash-ladder-client",
)

