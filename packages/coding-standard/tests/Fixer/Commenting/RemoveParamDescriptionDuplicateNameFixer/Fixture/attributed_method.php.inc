<?php

class Controller
{
    /**
     * @param int $leadId Lead ID
     */
    #[Route('/contact/{leadId}')]
    public function send($leadId)
    {}
}

?>
-----
<?php

class Controller
{
    /**
     * @param int $leadId
     */
    #[Route('/contact/{leadId}')]
    public function send($leadId)
    {}
}

?>
