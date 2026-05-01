import path from "node:path";

export default {
    mode: "production",
    entry: "./src/main.ts",
    output: {
        filename: "gotty-bundle.js",
        path: path.resolve(import.meta.dirname, "dist"),
    },
    devtool: "source-map",
    performance: {
        hints: false,
    },
    resolve: {
        extensions: [".ts", ".tsx", ".js"],
    },
    module: {
        rules: [
            {
                test: /\.tsx?$/,
                loader: "ts-loader",
                exclude: /node_modules/,
            },
        ],
    },
};
