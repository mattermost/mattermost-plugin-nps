// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Client4} from 'mattermost-redux/client';

import manifest from './manifest';

export class Client {
    constructor() {
        this.url = `/plugins/${manifest.id}/api/v1`;
    }

    connected = () => {
        return this.doFetch(`${this.url}/connected`, {method: 'POST'});
    }

    userWantsToGiveFeedback = () => {
        return this.doFetch(`${this.url}/give_feedback`, {method: 'POST'});
    }

    doFetch = async (url, options) => {
        if (!options.headers) {
            options.headers = {};
        }

        const opts = Client4.getOptions(options);

        try {
            const response = await fetch(url, opts);
            const data = await response.json();

            return {data};
        } catch (error) {
            return {error};
        }
    }
}
