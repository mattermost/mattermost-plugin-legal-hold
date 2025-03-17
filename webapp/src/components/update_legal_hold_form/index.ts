import {connect} from 'react-redux';
import {AnyAction, bindActionCreators, Dispatch} from 'redux';

import {getMissingProfilesByIds} from 'mattermost-redux/actions/users';

import {GlobalState} from 'mattermost-redux/types/store';
import {getUser} from 'mattermost-redux/selectors/entities/users';
import {getGroup} from 'mattermost-redux/selectors/entities/groups';

import {LegalHold} from '@/types';
import UpdateLegalHoldForm from '@/components/update_legal_hold_form/update_legal_hold_form';
import {getMissingGroupsByIds} from '@/components/legal_hold_table/index';

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

        const groups = ownProps.legalHold.group_ids.map((group_id) => getGroup(state, group_id));
        const users = ownProps.legalHold.user_ids.map((user_id) => getUser(state, user_id));
        return {
	    groups,
            users,
        };
    };
}

function mapDispatchToProps(dispatch: Dispatch<AnyAction>) {
    return {
        actions: bindActionCreators({
            getMissingProfilesByIds,
            getMissingGroupsByIds,
        }, dispatch),
    };
}

export default connect(makeMapStateToProps, mapDispatchToProps)(UpdateLegalHoldForm);
