import React from 'react';

import ConfirmFeedbackModal from './confirm_feedback_modal';

export default class Root extends React.PureComponent {
	render() {
		return (
			<React.Fragment>
				<ConfirmFeedbackModal/>
			</React.Fragment>
		);
	}
}