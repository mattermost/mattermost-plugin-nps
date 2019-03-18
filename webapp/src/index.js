import {id as pluginId} from './manifest';

import {Client} from './client';

export default class Plugin {
    constructor() {
        this.client = null;
    }

    initialize(/*registry, store*/) {
        this.client = new Client();

        this.client.connected();
    }
}

window.registerPlugin(pluginId, new Plugin());
