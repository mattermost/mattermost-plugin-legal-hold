import {useCallback} from 'react';

import {Combobox} from '@mattermost/compass-ui/components/combobox';
import type {Group} from '@mattermost/types/groups';

import Client from '@/client';
import useAsyncOptions from '@/hooks/use_async_options';

type Props = {
    label: string;
    groups: Group[];
    onChange: (groups: Group[]) => void;
};

const getValue = (group: Group) => group.id;

const getLabel = (group: Group) => {
    if (group.name && group.name !== group.display_name) {
        return `${group.display_name} - @${group.name}`;
    }
    return group.display_name;
};

// GroupsInput searches LDAP groups through the plugin API and lets the admin pick several of them.
const GroupsInput = ({label, groups, onChange}: Props) => {
    // The plugin answers null, not an empty list, when no group matches.
    const search = useCallback(async (term: string): Promise<Group[]> => (await Client.searchGroups(term)) ?? [], []);

    const {options, selectedOptions, selectedValues, loading, error, onInputChange, itemsForValues} = useAsyncOptions({
        search,
        getValue,
        getLabel,
        selected: groups,
        errorMessage: 'Could not search groups.',
    });

    return (
        <Combobox
            id='legal-hold-groups'
            label={label}
            multiple={true}
            filter={false}
            loading={loading}
            options={options}
            value={selectedValues}
            selectedOptions={selectedOptions}
            emptyMessage='Type to search for LDAP groups'
            message={error || undefined}
            onInputChange={onInputChange}
            onChange={(value) => onChange(itemsForValues(Array.isArray(value) ? value : []))}
        />
    );
};

export default GroupsInput;
