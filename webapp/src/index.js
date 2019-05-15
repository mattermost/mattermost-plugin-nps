import * as Actions from './actions';
import {Client} from './client';
import {POST_NPS_SURVEY} from './constants';
import Hooks from './hooks';
import {id as pluginId} from './manifest';
import reducer from './reducers';

import Root from './components/root';
import SurveyPost from './components/survey_post';

export default class Plugin {
    constructor() {
        this.registry = null;

        this.client = null;

        this.overrideScorePost = false;
        this.scorePostComponentId = '';
    }

    onWindowResize = () => {
        this.registerScorePostType();

        this.store.dispatch(Actions.windowResized(window.innerWidth));
    }

    registerScorePostType = () => {
        // The score post runs out of space slightly before switching to mobile view
        const overrideScorePost = window.innerWidth >= 800;

        if (overrideScorePost && !this.scorePostComponentId) {
            // There's enough space to use the custom score post, so register it
            this.scorePostComponentId = this.registry.registerPostTypeComponent(POST_NPS_SURVEY, SurveyPost);
        } else if (!overrideScorePost && this.scorePostComponentId) {
            // There's not enough space to use the custom score post, so remove it
            this.registry.unregisterPostTypeComponent(this.scorePostComponentId);
            this.scorePostComponentId = '';
        }
    }

    initialize(registry, store) {
        this.registry = registry;
        this.store = store;

        // Register reducer
        registry.registerReducer(reducer);

        // Register hooks
        const hooks = new Hooks(store);
        registry.registerMessageWillBePostedHook(hooks.messageWillBePosted);

        // Register components
        registry.registerRootComponent(Root);

        window.addEventListener('resize', this.onWindowResize);
        this.onWindowResize();

        // Initialize client
        this.client = new Client();
        store.dispatch(Actions.connected(this.client));
    }

    uninitialize() {
        window.removeEventListener('resize', this.onWindowResize);
    }
}

window.registerPlugin(pluginId, new Plugin());
