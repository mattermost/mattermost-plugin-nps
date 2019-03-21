import {General} from 'mattermost-redux/constants';
import {makeGetChannel} from 'mattermost-redux/selectors/entities/channels';
import {getUser} from 'mattermost-redux/selectors/entities/users';

import * as Actions from './actions';

export default class Hooks {
	constructor(store) {
		this.store = store;
	}

	messageWillBePosted = (post) => {
		const channel = makeGetChannel()(store.getState(), {id: post.channel_id});

		if (channel.type !== General.DM_CHANNEL) {
			console.log('not dm chanel', channel);
			return Promise.resolve({post});
		}

		// makeGetChannel passes the channel through completeDirectChannelInfo, so it has extra data that helps here
		const teammate = getUser(store.getState(), channel.teammate_id);

		if (teammate.username !== 'surveybot') {
			console.log('not surveybot', teammate);
			return Promise.resolve({post});
		}

		return new Promise((resolve) => {
			console.log('showing');
			const onConfirm = () => resolve({post});
			const onCancel = () => resolve({error: {message: 'error'}}); // TODO write an actual error

			this.store.dispatch(Actions.showConfirmationModal(onConfirm, onCancel));
		});
	}
}