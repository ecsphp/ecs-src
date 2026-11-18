<?php
require 'x.php';
?>
<ul>
    <?php foreach ($items as $item) { ?>
        <li>
            <?php
            if ($item) {
                echo $item;
            }
            ?>
        </li>
    <?php } ?>
</ul>
