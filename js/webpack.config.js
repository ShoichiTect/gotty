const path = require('path');

module.exports = {
    mode: 'production',
    entry: "./src/main.ts",
    output: {
        filename: "gotty-bundle.js",
        path: path.resolve(__dirname, 'dist'),
    },
    devtool: "source-map",
    resolve: {
        extensions: [".ts", ".tsx", ".js"],
    },
    module: {
        rules: [
            {
                test: /\.tsx?$/,
                loader: "ts-loader",
                exclude: /node_modules/
            }
        ]
    },
};
