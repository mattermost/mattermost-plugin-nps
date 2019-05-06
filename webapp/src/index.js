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
        this.client = new Client();

        this.registry = registry;

        window.addEventListener('resize', this.registerScorePostType);
        this.registerScorePostType();

        registry.registerRootComponent(Root);

        registry.registerReducer(reducer);

        const hooks = new Hooks(store);
        registry.registerMessageWillBePostedHook(hooks.messageWillBePosted);

        store.dispatch(Actions.connected(this.client));
    }

    uninitialize() {
        window.removeEventListener('resize', this.registerScorePostType);
    }
}

window.registerPlugin(pluginId, new Plugin());
