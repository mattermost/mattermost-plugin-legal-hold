import {TextInput} from '@mattermost/compass-ui/components/text-input';

import BaseSetting from './base_setting';

type Props = {
    id: string;
    name: string;
    helpText: string;
    onChange: (value: string) => void;
    value: string;
    disabled?: boolean;
};

const TextSetting = (props: Props) => {
    return (
        <BaseSetting
            id={props.id}
            name={props.name}
            helpText={props.helpText}
        >
            <TextInput
                id={props.id}
                data-testid={`${props.id}-input`}
                autoComplete='off'
                value={props.value}
                onChange={(e) => props.onChange(e.target.value)}
                disabled={props.disabled}
            />
        </BaseSetting>
    );
};

export default TextSetting;
