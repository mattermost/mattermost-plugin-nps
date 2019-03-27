import {id as pluginId} from './manifest';

export class Client {
    constructor() {
        this.url = `/plugins/${pluginId}/api/v1`;
    }

    connected = () => {
        return this.doFetch(`${this.url}/connected`, {
            method: 'POST',
            headers: {
                'X-Requested-With': 'XMLHttpRequest',
            },
        });
    }

    doFetch = async (url, options) => {
        try {
            const response = await fetch(url, options);
            const data = await response.json();

            return {data};
        } catch (error) {
            return {error};
        }
    }
}
