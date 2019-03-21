import {connect} from 'react-redux';

import {hideConfirmationModal} from '../../actions';
import {getConfirmationModalState} from '../../selectors';

import ConfirmFeedbackModal from './confirm_feedback_modal';

function mapStateToProps(state) {
	return {
		...getConfirmationModalState(state),
	};
}

const mapDispatchToProps = {
	hideConfirmationModal,
};

export default connect(mapStateToProps, mapDispatchToProps)(ConfirmFeedbackModal);