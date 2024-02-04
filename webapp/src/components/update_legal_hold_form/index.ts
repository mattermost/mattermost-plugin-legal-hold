import {connect} from 'react-redux';
import {AnyAction, bindActionCreators, Dispatch} from 'redux';

import {getMissingProfilesByIds} from 'mattermost-redux/actions/users';

import {GlobalState} from 'mattermost-redux/types/store';
import {getUser} from 'mattermost-redux/selectors/entities/users';

import {LegalHold} from '@/types';
import UpdateLegalHoldForm from '@/components/update_legal_hold_form/update_legal_hold_form';

type OwnProps = {
    legalHold: LegalHold|null;
}

function makeMapStateToProps() {
    return (state: GlobalState, ownProps: OwnProps) => {
        if (ownProps.legalHold === null || ownProps.legalHold.user_ids === null) {
            return {
                users: [],
            };
        }

        const users = ownProps.legalHold.user_ids.map((user_id) => getUser(state, user_id));
        return {
            users,
        };
    };
}

function mapDispatchToProps(dispatch: Dispatch<AnyAction>) {
    return {
        actions: bindActionCreators({
            getMissingProfilesByIds,
        }, dispatch),
    };
}

export default connect(makeMapStateToProps, mapDispatchToProps)(UpdateLegalHoldForm);
