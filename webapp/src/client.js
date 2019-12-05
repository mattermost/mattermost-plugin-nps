import {Client4} from 'mattermost-redux/client';

import {id as pluginId} from './manifest';

export class Client {
    constructor() {
        this.url = `/plugins/${pluginId}/api/v1`;
    }

    connected = () => {
        return this.doFetch(`${this.url}/connected`, {method: 'POST'});
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
