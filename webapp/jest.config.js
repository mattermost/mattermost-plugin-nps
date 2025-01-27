// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

module.exports = {
    clearMocks: true,
    globals: {
        ReactBootstrap: {},
    },
    setupFilesAfterEnv: [
        '<rootDir>/test_setup.js',
    ],
};
