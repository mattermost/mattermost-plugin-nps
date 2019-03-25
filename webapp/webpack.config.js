var path = require('path');

module.exports = {
    entry: [
        './src/index.js',
    ],
    resolve: {
        modules: [
            'src',
            'node_modules',
        ],
        extensions: ['*', '.js', '.jsx'],
    },
    module: {
        rules: [
            {
                test: /\.(js|jsx)$/,
                exclude: /node_modules/,
                use: {
                    loader: 'babel-loader',
                    options: {
                        presets: ['env', 'react'],
                        plugins: [
                            'transform-class-properties',
                            'transform-object-rest-spread',
                        ],
                    },
                },
            },
        ],
    },
    externals: {
        'prop-types': 'PropTypes',
        react: 'React',
        'react-bootstrap': 'ReactBootstrap',
        'react-dom': 'ReactDOM',
        'react-redux': 'ReactRedux',
        redux: 'Redux',
    },
    output: {
        path: path.join(__dirname, '/dist'),
        publicPath: '/',
        filename: 'main.js',
    },
};
