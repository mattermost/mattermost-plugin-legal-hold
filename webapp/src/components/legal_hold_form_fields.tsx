import {Button} from '@mattermost/compass-ui/components/button';
import {Checkbox} from '@mattermost/compass-ui/components/checkbox';
import {DateRangePicker} from '@mattermost/compass-ui/components/date-range-picker';
import {SectionNotice} from '@mattermost/compass-ui/components/section-notice';
import {TextInput} from '@mattermost/compass-ui/components/text-input';
import type {Group} from '@mattermost/types/groups';
import type {UserProfile} from '@mattermost/types/users';

import GroupsInput from '@/components/groups_input';
import requiredLabel from '@/components/required_label';
import UsersInput from '@/components/users_input';

import './legal_hold_form.css';

type Props = {
    displayName: string;
    onDisplayNameChange: (value: string) => void;
    users: UserProfile[];
    onUsersChange: (users: UserProfile[]) => void;
    groups: Group[];
    onGroupsChange: (groups: Group[]) => void;
    includePublicChannels: boolean;
    onIncludePublicChannelsChange: (value: boolean) => void;
    publicChannelsNotice: string;
    startsAt: string;
    startsAtLabel: string;

    // When set, the start date is editable. A legal hold keeps its start date after it is created.
    onStartsAtChange?: (value: string) => void;
    endsAt: string;
    onEndsAtChange: (value: string) => void;
};

// The fields that the create and update forms share.
const LegalHoldFormFields = (props: Props) => {
    return (
        <>
            <TextInput
                id='legal-hold-name'
                label={requiredLabel('Name')}
                required={true}
                autoComplete='off'
                maxLength={64}
                showCharacterCount={true}
                value={props.displayName}
                onChange={(e) => props.onDisplayNameChange(e.target.value)}
            />
            <UsersInput
                label='Users'
                users={props.users}
                onChange={props.onUsersChange}
            />
            <GroupsInput
                label='LDAP Groups'
                groups={props.groups}
                onChange={props.onGroupsChange}
            />
            <Checkbox
                id='legal-hold-include-public-channels'
                checked={props.includePublicChannels}
                onChange={(e) => props.onIncludePublicChannelsChange(e.target.checked)}
            >
                {'Include public channels'}
            </Checkbox>
            {props.includePublicChannels && (
                <SectionNotice
                    type='warning'
                    title={props.publicChannelsNotice}
                />
            )}
            <div className='legal-hold-form__dates'>
                <DateRangePicker
                    id='legal-hold-starts-at'
                    mode='date'
                    label={requiredLabel(props.startsAtLabel)}
                    value={props.startsAt}
                    onChange={props.onStartsAtChange}
                    disabled={!props.onStartsAtChange}
                />
                <div className='legal-hold-form__date'>
                    <DateRangePicker
                        id='legal-hold-ends-at'
                        mode='date'
                        label='Ending at'
                        value={props.endsAt}
                        onChange={props.onEndsAtChange}
                    />
                    {props.endsAt !== '' && (
                        <Button
                            type='button'
                            emphasis='link'
                            size='small'
                            onClick={() => props.onEndsAtChange('')}
                        >
                            {'Clear end date'}
                        </Button>
                    )}
                </div>
            </div>
        </>
    );
};

export default LegalHoldFormFields;
