import * as Actions from './actions';

import {Client} from './client';

import Root from './components/root';

import Hooks from './hooks';

import {id as pluginId} from './manifest';

import reducer from './reducers';

export default class Plugin {
    constructor() {
        this.client = null;
    }

    initialize(registry, store) {
        this.client = new Client();

        registry.registerRootComponent(Root);

        registry.registerReducer(reducer);

        const hooks = new Hooks(store);
        registry.registerMessageWillBePostedHook(hooks.messageWillBePosted);

        store.dispatch(Actions.connected(this.client));
    }
}

window.registerPlugin(pluginId, new Plugin());
